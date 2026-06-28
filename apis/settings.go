package apis

import (
	"net/http"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/forms"
	"github.com/zhenruyan/postgrebase/models/settings"
)

// bindSettingsApi registers the settings api endpoints.
func bindSettingsApi(app core.App, rg *echo.Group) {
	api := settingsApi{app: app}

	subGroup := rg.Group("/settings", ActivityLogger(app), RequireAdminAuth())
	subGroup.GET("", api.list)
	subGroup.PATCH("", api.set)
	subGroup.POST("/test/s3", api.testS3)
	subGroup.POST("/test/webdav", api.testWebDAV)
	subGroup.POST("/test/email", api.testEmail)
	subGroup.POST("/apple/generate-client-secret", api.generateAppleClientSecret)
}

type settingsApi struct {
	app core.App
}

func (api *settingsApi) list(c echo.Context) error {
	settings, err := api.app.Settings().RedactClone()
	if err != nil {
		return NewBadRequestError("", err)
	}

	event := new(core.SettingsListEvent)
	event.HttpContext = c
	event.RedactedSettings = settings

	return api.app.OnSettingsListRequest().Trigger(event, func(e *core.SettingsListEvent) error {
		if e.HttpContext.Response().Committed {
			return nil
		}

		return e.HttpContext.JSON(http.StatusOK, e.RedactedSettings)
	})
}

func (api *settingsApi) set(c echo.Context) error {
	form := forms.NewSettingsUpsert(api.app)

	// load request
	if err := c.Bind(form); err != nil {
		return NewBadRequestError("An error occurred while loading the submitted data.", err)
	}

	// Preserve redacted agent provider API keys: the admin UI sends masked
	// secrets that are stripped client-side, so an empty incoming key for an
	// existing provider must keep the previously stored value.
	restoreAgentProviderSecrets(api.app.Settings(), form.Settings)

	event := new(core.SettingsUpdateEvent)
	event.HttpContext = c
	event.OldSettings = api.app.Settings()

	// update the settings
	return form.Submit(func(next forms.InterceptorNextFunc[*settings.Settings]) forms.InterceptorNextFunc[*settings.Settings] {
		return func(s *settings.Settings) error {
			event.NewSettings = s

			return api.app.OnSettingsBeforeUpdateRequest().Trigger(event, func(e *core.SettingsUpdateEvent) error {
				if err := next(e.NewSettings); err != nil {
					return NewBadRequestError("An error occurred while submitting the form.", err)
				}

				return api.app.OnSettingsAfterUpdateRequest().Trigger(event, func(e *core.SettingsUpdateEvent) error {
					if e.HttpContext.Response().Committed {
						return nil
					}

					redactedSettings, err := api.app.Settings().RedactClone()
					if err != nil {
						return NewBadRequestError("", err)
					}

					return e.HttpContext.JSON(http.StatusOK, redactedSettings)
				})
			})
		}
	})
}

func (api *settingsApi) testS3(c echo.Context) error {
	form := forms.NewTestS3Filesystem(api.app)

	// load request
	if err := c.Bind(form); err != nil {
		return NewBadRequestError("An error occurred while loading the submitted data.", err)
	}

	// send
	if err := form.Submit(); err != nil {
		// form error
		if fErr, ok := err.(validation.Errors); ok {
			return NewBadRequestError("Failed to test the S3 filesystem.", fErr)
		}

		// mailer error
		return NewBadRequestError("Failed to test the S3 filesystem. Raw error: \n"+err.Error(), nil)
	}

	return c.NoContent(http.StatusNoContent)
}

func (api *settingsApi) testWebDAV(c echo.Context) error {
	form := forms.NewTestWebDAVFilesystem(api.app)

	// load request
	if err := c.Bind(form); err != nil {
		return NewBadRequestError("An error occurred while loading the submitted data.", err)
	}

	// send
	if err := form.Submit(); err != nil {
		// form error
		if fErr, ok := err.(validation.Errors); ok {
			return NewBadRequestError("Failed to test the WebDAV filesystem.", fErr)
		}

		// mailer error
		return NewBadRequestError("Failed to test the WebDAV filesystem. Raw error: \n"+err.Error(), nil)
	}

	return c.NoContent(http.StatusNoContent)
}

func (api *settingsApi) testEmail(c echo.Context) error {
	form := forms.NewTestEmailSend(api.app)

	// load request
	if err := c.Bind(form); err != nil {
		return NewBadRequestError("An error occurred while loading the submitted data.", err)
	}

	// send
	if err := form.Submit(); err != nil {
		// form error
		if fErr, ok := err.(validation.Errors); ok {
			return NewBadRequestError("Failed to send the test email.", fErr)
		}

		// mailer error
		return NewBadRequestError("Failed to send the test email. Raw error: \n"+err.Error(), nil)
	}

	return c.NoContent(http.StatusNoContent)
}

func (api *settingsApi) generateAppleClientSecret(c echo.Context) error {
	form := forms.NewAppleClientSecretCreate(api.app)

	// load request
	if err := c.Bind(form); err != nil {
		return NewBadRequestError("An error occurred while loading the submitted data.", err)
	}

	// generate
	secret, err := form.Submit()
	if err != nil {
		// form error
		if fErr, ok := err.(validation.Errors); ok {
			return NewBadRequestError("Invalid client secret data.", fErr)
		}

		// secret generation error
		return NewBadRequestError("Failed to generate client secret. Raw error: \n"+err.Error(), nil)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"secret": secret,
	})
}

// restoreAgentProviderSecrets copies non-empty API keys from the old settings
// into incoming providers (matched by id) whenever the incoming key is empty or
// still carries the redaction mask (the admin UI may echo back the masked
// value instead of stripping it, which would otherwise overwrite the real key).
func restoreAgentProviderSecrets(old *settings.Settings, next *settings.Settings) {
	if old == nil || next == nil {
		return
	}
	existing := map[string]string{}
	for _, p := range old.Agents.Providers {
		if p.Id != "" && p.ApiKey != "" {
			existing[p.Id] = p.ApiKey
		}
	}
	for i := range next.Agents.Providers {
		key := strings.TrimSpace(next.Agents.Providers[i].ApiKey)
		if key == "" || key == settings.SecretMask {
			if old, ok := existing[next.Agents.Providers[i].Id]; ok {
				next.Agents.Providers[i].ApiKey = old
			} else if key == settings.SecretMask {
				// no previous value to restore: drop the mask so it is not
				// persisted as a literal api key
				next.Agents.Providers[i].ApiKey = ""
			}
		}
	}

	existingEmbeddings := map[string]string{}
	for _, p := range old.Agents.Embedding.Providers {
		if p.Id != "" && p.ApiKey != "" {
			existingEmbeddings[p.Id] = p.ApiKey
		}
	}
	for i := range next.Agents.Embedding.Providers {
		key := strings.TrimSpace(next.Agents.Embedding.Providers[i].ApiKey)
		if key == "" || key == settings.SecretMask {
			if old, ok := existingEmbeddings[next.Agents.Embedding.Providers[i].Id]; ok {
				next.Agents.Embedding.Providers[i].ApiKey = old
			} else if key == settings.SecretMask {
				next.Agents.Embedding.Providers[i].ApiKey = ""
			}
		}
	}
}
