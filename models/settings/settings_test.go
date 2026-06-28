package settings_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/zhenruyan/postgrebase/models/settings"
	"github.com/zhenruyan/postgrebase/tools/auth"
	"github.com/zhenruyan/postgrebase/tools/mailer"
)

func TestSettingsValidate(t *testing.T) {
	s := settings.New()

	// set invalid settings data
	s.Meta.AppName = ""
	s.Logs.MaxDays = -10
	s.Smtp.Enabled = true
	s.Smtp.Host = ""
	s.S3.Enabled = true
	s.S3.Endpoint = "invalid"
	s.AdminAuthToken.Duration = -10
	s.AdminPasswordResetToken.Duration = -10
	s.AdminFileToken.Duration = -10
	s.RecordAuthToken.Duration = -10
	s.RecordPasswordResetToken.Duration = -10
	s.RecordEmailChangeToken.Duration = -10
	s.RecordVerificationToken.Duration = -10
	s.RecordFileToken.Duration = -10
	s.GoogleAuth.Enabled = true
	s.GoogleAuth.ClientId = ""
	s.FacebookAuth.Enabled = true
	s.FacebookAuth.ClientId = ""
	s.GithubAuth.Enabled = true
	s.GithubAuth.ClientId = ""
	s.GitlabAuth.Enabled = true
	s.GitlabAuth.ClientId = ""
	s.DiscordAuth.Enabled = true
	s.DiscordAuth.ClientId = ""
	s.TwitterAuth.Enabled = true
	s.TwitterAuth.ClientId = ""
	s.MicrosoftAuth.Enabled = true
	s.MicrosoftAuth.ClientId = ""
	s.SpotifyAuth.Enabled = true
	s.SpotifyAuth.ClientId = ""
	s.KakaoAuth.Enabled = true
	s.KakaoAuth.ClientId = ""
	s.TwitchAuth.Enabled = true
	s.TwitchAuth.ClientId = ""
	s.StravaAuth.Enabled = true
	s.StravaAuth.ClientId = ""
	s.GiteeAuth.Enabled = true
	s.GiteeAuth.ClientId = ""
	s.LivechatAuth.Enabled = true
	s.LivechatAuth.ClientId = ""
	s.GiteaAuth.Enabled = true
	s.GiteaAuth.ClientId = ""
	s.OIDCAuth.Enabled = true
	s.OIDCAuth.ClientId = ""
	s.OIDC2Auth.Enabled = true
	s.OIDC2Auth.ClientId = ""
	s.OIDC3Auth.Enabled = true
	s.OIDC3Auth.ClientId = ""
	s.AppleAuth.Enabled = true
	s.AppleAuth.ClientId = ""
	s.InstagramAuth.Enabled = true
	s.InstagramAuth.ClientId = ""
	s.VKAuth.Enabled = true
	s.VKAuth.ClientId = ""
	s.YandexAuth.Enabled = true
	s.YandexAuth.ClientId = ""

	// check if Validate() is triggering the members validate methods.
	err := s.Validate()
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}

	expectations := []string{
		`"meta":{`,
		`"logs":{`,
		`"smtp":{`,
		`"s3":{`,
		`"adminAuthToken":{`,
		`"adminPasswordResetToken":{`,
		`"adminFileToken":{`,
		`"recordAuthToken":{`,
		`"recordPasswordResetToken":{`,
		`"recordEmailChangeToken":{`,
		`"recordVerificationToken":{`,
		`"recordFileToken":{`,
		`"googleAuth":{`,
		`"facebookAuth":{`,
		`"githubAuth":{`,
		`"gitlabAuth":{`,
		`"discordAuth":{`,
		`"twitterAuth":{`,
		`"microsoftAuth":{`,
		`"spotifyAuth":{`,
		`"kakaoAuth":{`,
		`"twitchAuth":{`,
		`"stravaAuth":{`,
		`"giteeAuth":{`,
		`"livechatAuth":{`,
		`"giteaAuth":{`,
		`"oidcAuth":{`,
		`"oidc2Auth":{`,
		`"oidc3Auth":{`,
		`"appleAuth":{`,
		`"instagramAuth":{`,
		`"vkAuth":{`,
		`"yandexAuth":{`,
	}

	errBytes, _ := json.Marshal(err)
	jsonErr := string(errBytes)
	for _, expected := range expectations {
		if !strings.Contains(jsonErr, expected) {
			t.Errorf("Expected error key %s in %v", expected, jsonErr)
		}
	}
}

func TestSettingsMerge(t *testing.T) {
	s1 := settings.New()
	s1.Meta.AppUrl = "old_app_url"

	s2 := settings.New()
	s2.Meta.AppName = "test"
	s2.Logs.MaxDays = 123
	s2.Smtp.Host = "test"
	s2.Smtp.Enabled = true
	s2.S3.Enabled = true
	s2.S3.Endpoint = "test"
	s2.Backups.Cron = "* * * * *"
	s2.AdminAuthToken.Duration = 1
	s2.AdminPasswordResetToken.Duration = 2
	s2.AdminFileToken.Duration = 2
	s2.RecordAuthToken.Duration = 3
	s2.RecordPasswordResetToken.Duration = 4
	s2.RecordEmailChangeToken.Duration = 5
	s2.RecordVerificationToken.Duration = 6
	s2.RecordFileToken.Duration = 7
	s2.GoogleAuth.Enabled = true
	s2.GoogleAuth.ClientId = "google_test"
	s2.FacebookAuth.Enabled = true
	s2.FacebookAuth.ClientId = "facebook_test"
	s2.GithubAuth.Enabled = true
	s2.GithubAuth.ClientId = "github_test"
	s2.GitlabAuth.Enabled = true
	s2.GitlabAuth.ClientId = "gitlab_test"
	s2.DiscordAuth.Enabled = true
	s2.DiscordAuth.ClientId = "discord_test"
	s2.TwitterAuth.Enabled = true
	s2.TwitterAuth.ClientId = "twitter_test"
	s2.MicrosoftAuth.Enabled = true
	s2.MicrosoftAuth.ClientId = "microsoft_test"
	s2.SpotifyAuth.Enabled = true
	s2.SpotifyAuth.ClientId = "spotify_test"
	s2.KakaoAuth.Enabled = true
	s2.KakaoAuth.ClientId = "kakao_test"
	s2.TwitchAuth.Enabled = true
	s2.TwitchAuth.ClientId = "twitch_test"
	s2.StravaAuth.Enabled = true
	s2.StravaAuth.ClientId = "strava_test"
	s2.GiteeAuth.Enabled = true
	s2.GiteeAuth.ClientId = "gitee_test"
	s2.LivechatAuth.Enabled = true
	s2.LivechatAuth.ClientId = "livechat_test"
	s2.GiteaAuth.Enabled = true
	s2.GiteaAuth.ClientId = "gitea_test"
	s2.OIDCAuth.Enabled = true
	s2.OIDCAuth.ClientId = "oidc_test"
	s2.OIDC2Auth.Enabled = true
	s2.OIDC2Auth.ClientId = "oidc2_test"
	s2.OIDC3Auth.Enabled = true
	s2.OIDC3Auth.ClientId = "oidc3_test"
	s2.AppleAuth.Enabled = true
	s2.AppleAuth.ClientId = "apple_test"
	s2.InstagramAuth.Enabled = true
	s2.InstagramAuth.ClientId = "instagram_test"
	s2.VKAuth.Enabled = true
	s2.VKAuth.ClientId = "vk_test"
	s2.YandexAuth.Enabled = true
	s2.YandexAuth.ClientId = "yandex_test"
	s2.Agents.Enabled = true
	s2.Agents.DefaultProvider = "openai-main"
	s2.Agents.DefaultModel = "gpt-4.1"
	s2.Agents.AllowedTools = []string{"data.query"}
	s2.Agents.Providers = []settings.AgentProviderConfig{
		{
			Id:           "openai-main",
			Vendor:       "openai",
			BaseUrl:      "https://api.openai.com/v1",
			Enabled:      true,
			DefaultModel: "gpt-4.1",
			Models: []settings.AgentProviderModel{
				{
					Name:            "gpt-4.1",
					ProviderModelId: "gpt-4.1",
					Enabled:         true,
				},
			},
		},
	}

	if err := s1.Merge(s2); err != nil {
		t.Fatal(err)
	}

	s1Encoded, err := json.Marshal(s1)
	if err != nil {
		t.Fatal(err)
	}

	s2Encoded, err := json.Marshal(s2)
	if err != nil {
		t.Fatal(err)
	}

	if string(s1Encoded) != string(s2Encoded) {
		t.Fatalf("Expected the same serialization, got %v VS %v", string(s1Encoded), string(s2Encoded))
	}
}

func TestAgentProviderVendorIsFreeForm(t *testing.T) {
	cfg := settings.AgentProviderConfig{
		Id:      "gitee-main",
		Vendor:  "gitee",
		Api:     "openai-chat",
		BaseUrl: "https://example.com/v1",
		Enabled: true,
		Models: []settings.AgentProviderModel{
			{Name: "kimi2.5", ProviderModelId: "kimi2.5", Enabled: true},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("vendor should be free-form now: %v", err)
	}
}

func TestAgentProviderVendorValidation(t *testing.T) {
	cases := []struct {
		name   string
		vendor string
	}{
		{name: "openai", vendor: "openai"},
		{name: "anthropic", vendor: "anthropic"},
		{name: "gemini", vendor: "google-gemini"},
		{name: "vertex", vendor: "google-vertex"},
		{name: "custom", vendor: "gitee"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := settings.AgentProviderConfig{
				Id:      "p1",
				Vendor:  tc.vendor,
				Api:     "openai-chat",
				BaseUrl: "https://example.com/v1",
				Models: []settings.AgentProviderModel{
					{Name: "m1", ProviderModelId: "m1", Enabled: true},
				},
			}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestAgentProviderApiValidation(t *testing.T) {
	cases := []struct {
		name    string
		api     string
		wantErr bool
	}{
		{name: "chat", api: "openai-chat", wantErr: false},
		{name: "responses", api: "openai-responses", wantErr: false},
		{name: "anthropic", api: "anthropic-messages", wantErr: false},
		{name: "gemini", api: "google-gemini", wantErr: false},
		{name: "vertex", api: "google-vertex", wantErr: false},
		{name: "invalid", api: "openai-compatible", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := settings.AgentProviderConfig{
				Id:      "p1",
				Vendor:  "openai",
				Api:     tc.api,
				BaseUrl: "https://example.com/v1",
				Models: []settings.AgentProviderModel{
					{Name: "m1", ProviderModelId: "m1", Enabled: true},
				},
			}
			err := cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestSettingsClone(t *testing.T) {
	s1 := settings.New()

	s2, err := s1.Clone()
	if err != nil {
		t.Fatal(err)
	}

	s1Bytes, err := json.Marshal(s1)
	if err != nil {
		t.Fatal(err)
	}

	s2Bytes, err := json.Marshal(s2)
	if err != nil {
		t.Fatal(err)
	}

	if string(s1Bytes) != string(s2Bytes) {
		t.Fatalf("Expected equivalent serialization, got %v VS %v", string(s1Bytes), string(s2Bytes))
	}

	// verify that it is a deep copy
	s1.Meta.AppName = "new"
	if s1.Meta.AppName == s2.Meta.AppName {
		t.Fatalf("Expected s1 and s2 to have different Meta.AppName, got %s", s1.Meta.AppName)
	}
}

func TestSettingsRedactClone(t *testing.T) {
	testSecret := "test_secret"

	s1 := settings.New()

	// control fields
	s1.Meta.AppName = "test123"
	s1.Agents.Enabled = true
	s1.Agents.DefaultProvider = "openai-main"
	s1.Agents.DefaultModel = "gpt-4.1"
	s1.Agents.AllowSchemaChange = true
	s1.Agents.AllowedTools = []string{"data.query", "dataset.preview"}
	s1.Agents.Providers = []settings.AgentProviderConfig{
		{
			Id:           "openai-main",
			Vendor:       "openai",
			BaseUrl:      "https://api.openai.com/v1",
			ApiKey:       testSecret,
			Enabled:      true,
			DefaultModel: "gpt-4.1",
			Models: []settings.AgentProviderModel{
				{
					Name:             "gpt-4.1",
					ProviderModelId:  "gpt-4.1",
					SupportsVision:   true,
					SupportsToolUse:  true,
					SupportsDocument: false,
					Enabled:          true,
				},
			},
		},
	}
	s1.Agents.Embedding = settings.AgentEmbeddingConfig{
		Enabled:      true,
		DefaultModel: "text-embedding-3-small",
		Providers: []settings.AgentEmbeddingProviderConfig{
			{
				Id:      "openai-embedding",
				Vendor:  "openai",
				Api:     "openai-embeddings",
				BaseUrl: "https://api.openai.com/v1",
				ApiKey:  testSecret,
				Enabled: true,
				Models: []settings.AgentEmbeddingModel{
					{Name: "text-embedding-3-small", ProviderModelId: "text-embedding-3-small", Enabled: true},
				},
			},
		},
	}

	// secrets
	s1.Smtp.Password = testSecret
	s1.S3.Secret = testSecret
	s1.Backups.S3.Secret = testSecret
	s1.AdminAuthToken.Secret = testSecret
	s1.AdminPasswordResetToken.Secret = testSecret
	s1.AdminFileToken.Secret = testSecret
	s1.RecordAuthToken.Secret = testSecret
	s1.RecordPasswordResetToken.Secret = testSecret
	s1.RecordEmailChangeToken.Secret = testSecret
	s1.RecordVerificationToken.Secret = testSecret
	s1.RecordFileToken.Secret = testSecret
	s1.GoogleAuth.ClientSecret = testSecret
	s1.FacebookAuth.ClientSecret = testSecret
	s1.GithubAuth.ClientSecret = testSecret
	s1.GitlabAuth.ClientSecret = testSecret
	s1.DiscordAuth.ClientSecret = testSecret
	s1.TwitterAuth.ClientSecret = testSecret
	s1.MicrosoftAuth.ClientSecret = testSecret
	s1.SpotifyAuth.ClientSecret = testSecret
	s1.KakaoAuth.ClientSecret = testSecret
	s1.TwitchAuth.ClientSecret = testSecret
	s1.StravaAuth.ClientSecret = testSecret
	s1.GiteeAuth.ClientSecret = testSecret
	s1.LivechatAuth.ClientSecret = testSecret
	s1.GiteaAuth.ClientSecret = testSecret
	s1.OIDCAuth.ClientSecret = testSecret
	s1.OIDC2Auth.ClientSecret = testSecret
	s1.OIDC3Auth.ClientSecret = testSecret
	s1.AppleAuth.ClientSecret = testSecret
	s1.InstagramAuth.ClientSecret = testSecret
	s1.VKAuth.ClientSecret = testSecret
	s1.YandexAuth.ClientSecret = testSecret

	s1Bytes, err := json.Marshal(s1)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := s1.RedactClone()
	if err != nil {
		t.Fatal(err)
	}

	s2Bytes, err := json.Marshal(s2)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(s1Bytes, s2Bytes) {
		t.Fatalf("Expected the 2 settings to differ, got \n%s", s2Bytes)
	}

	if strings.Contains(string(s2Bytes), testSecret) {
		t.Fatalf("Expected %q secret to be replaced with mask, got \n%s", testSecret, s2Bytes)
	}

	if !strings.Contains(string(s2Bytes), settings.SecretMask) {
		t.Fatalf("Expected the secrets to be replaced with the secret mask, got \n%s", s2Bytes)
	}

	if !strings.Contains(string(s2Bytes), `"appName":"test123"`) {
		t.Fatalf("Missing control field in \n%s", s2Bytes)
	}

	if !strings.Contains(string(s2Bytes), `"defaultProvider":"openai-main"`) {
		t.Fatalf("Missing agent settings in \n%s", s2Bytes)
	}

	if strings.Contains(string(s2Bytes), "sk-test") {
		t.Fatalf("Expected agent api key to be masked, got \n%s", s2Bytes)
	}
}

func TestNamedAuthProviderConfigs(t *testing.T) {
	s := settings.New()

	s.GoogleAuth.ClientId = "google_test"
	s.FacebookAuth.ClientId = "facebook_test"
	s.GithubAuth.ClientId = "github_test"
	s.GitlabAuth.ClientId = "gitlab_test"
	s.GitlabAuth.Enabled = true // control
	s.DiscordAuth.ClientId = "discord_test"
	s.TwitterAuth.ClientId = "twitter_test"
	s.MicrosoftAuth.ClientId = "microsoft_test"
	s.SpotifyAuth.ClientId = "spotify_test"
	s.KakaoAuth.ClientId = "kakao_test"
	s.TwitchAuth.ClientId = "twitch_test"
	s.StravaAuth.ClientId = "strava_test"
	s.GiteeAuth.ClientId = "gitee_test"
	s.LivechatAuth.ClientId = "livechat_test"
	s.GiteaAuth.ClientId = "gitea_test"
	s.OIDCAuth.ClientId = "oidc_test"
	s.OIDC2Auth.ClientId = "oidc2_test"
	s.OIDC3Auth.ClientId = "oidc3_test"
	s.AppleAuth.ClientId = "apple_test"
	s.InstagramAuth.ClientId = "instagram_test"
	s.VKAuth.ClientId = "vk_test"
	s.YandexAuth.ClientId = "yandex_test"

	result := s.NamedAuthProviderConfigs()

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	encodedStr := string(encoded)

	expectedParts := []string{
		`"discord":{"enabled":false,"clientId":"discord_test"`,
		`"facebook":{"enabled":false,"clientId":"facebook_test"`,
		`"github":{"enabled":false,"clientId":"github_test"`,
		`"gitlab":{"enabled":true,"clientId":"gitlab_test"`,
		`"google":{"enabled":false,"clientId":"google_test"`,
		`"microsoft":{"enabled":false,"clientId":"microsoft_test"`,
		`"spotify":{"enabled":false,"clientId":"spotify_test"`,
		`"twitter":{"enabled":false,"clientId":"twitter_test"`,
		`"kakao":{"enabled":false,"clientId":"kakao_test"`,
		`"twitch":{"enabled":false,"clientId":"twitch_test"`,
		`"strava":{"enabled":false,"clientId":"strava_test"`,
		`"gitee":{"enabled":false,"clientId":"gitee_test"`,
		`"livechat":{"enabled":false,"clientId":"livechat_test"`,
		`"gitea":{"enabled":false,"clientId":"gitea_test"`,
		`"oidc":{"enabled":false,"clientId":"oidc_test"`,
		`"oidc2":{"enabled":false,"clientId":"oidc2_test"`,
		`"oidc3":{"enabled":false,"clientId":"oidc3_test"`,
		`"apple":{"enabled":false,"clientId":"apple_test"`,
		`"instagram":{"enabled":false,"clientId":"instagram_test"`,
		`"vk":{"enabled":false,"clientId":"vk_test"`,
		`"yandex":{"enabled":false,"clientId":"yandex_test"`,
	}
	for _, p := range expectedParts {
		if !strings.Contains(encodedStr, p) {
			t.Fatalf("Expected \n%s \nin \n%s", p, encodedStr)
		}
	}
}

func TestTokenConfigValidate(t *testing.T) {
	scenarios := []struct {
		config      settings.TokenConfig
		expectError bool
	}{
		// zero values
		{
			settings.TokenConfig{},
			true,
		},
		// invalid data
		{
			settings.TokenConfig{
				Secret:   strings.Repeat("a", 5),
				Duration: 4,
			},
			true,
		},
		// valid secret but invalid duration
		{
			settings.TokenConfig{
				Secret:   strings.Repeat("a", 30),
				Duration: 63072000 + 1,
			},
			true,
		},
		// valid data
		{
			settings.TokenConfig{
				Secret:   strings.Repeat("a", 30),
				Duration: 100,
			},
			false,
		},
	}

	for i, scenario := range scenarios {
		result := scenario.config.Validate()

		if result != nil && !scenario.expectError {
			t.Errorf("(%d) Didn't expect error, got %v", i, result)
		}

		if result == nil && scenario.expectError {
			t.Errorf("(%d) Expected error, got nil", i)
		}
	}
}

func TestSmtpConfigValidate(t *testing.T) {
	scenarios := []struct {
		config      settings.SmtpConfig
		expectError bool
	}{
		// zero values (disabled)
		{
			settings.SmtpConfig{},
			false,
		},
		// zero values (enabled)
		{
			settings.SmtpConfig{Enabled: true},
			true,
		},
		// invalid data
		{
			settings.SmtpConfig{
				Enabled: true,
				Host:    "test:test:test",
				Port:    -10,
			},
			true,
		},
		// invalid auth method
		{
			settings.SmtpConfig{
				Enabled:    true,
				Host:       "example.com",
				Port:       100,
				AuthMethod: "example",
			},
			true,
		},
		// valid data (no explicit auth method)
		{
			settings.SmtpConfig{
				Enabled: true,
				Host:    "example.com",
				Port:    100,
				Tls:     true,
			},
			false,
		},
		// valid data (explicit auth method - login)
		{
			settings.SmtpConfig{
				Enabled:    true,
				Host:       "example.com",
				Port:       100,
				AuthMethod: mailer.SmtpAuthLogin,
			},
			false,
		},
	}

	for i, scenario := range scenarios {
		result := scenario.config.Validate()

		if result != nil && !scenario.expectError {
			t.Errorf("(%d) Didn't expect error, got %v", i, result)
		}

		if result == nil && scenario.expectError {
			t.Errorf("(%d) Expected error, got nil", i)
		}
	}
}

func TestS3ConfigValidate(t *testing.T) {
	scenarios := []struct {
		config      settings.S3Config
		expectError bool
	}{
		// zero values (disabled)
		{
			settings.S3Config{},
			false,
		},
		// zero values (enabled)
		{
			settings.S3Config{Enabled: true},
			true,
		},
		// invalid data
		{
			settings.S3Config{
				Enabled:  true,
				Endpoint: "test:test:test",
			},
			true,
		},
		// valid data (url endpoint)
		{
			settings.S3Config{
				Enabled:   true,
				Endpoint:  "https://localhost:8090",
				Bucket:    "test",
				Region:    "test",
				AccessKey: "test",
				Secret:    "test",
			},
			false,
		},
		// valid data (hostname endpoint)
		{
			settings.S3Config{
				Enabled:   true,
				Endpoint:  "example.com",
				Bucket:    "test",
				Region:    "test",
				AccessKey: "test",
				Secret:    "test",
			},
			false,
		},
	}

	for i, scenario := range scenarios {
		result := scenario.config.Validate()

		if result != nil && !scenario.expectError {
			t.Errorf("(%d) Didn't expect error, got %v", i, result)
		}

		if result == nil && scenario.expectError {
			t.Errorf("(%d) Expected error, got nil", i)
		}
	}
}

func TestMetaConfigValidate(t *testing.T) {
	invalidTemplate := settings.EmailTemplate{
		Subject:   "test",
		ActionUrl: "test",
		Body:      "test",
	}

	noPlaceholdersTemplate := settings.EmailTemplate{
		Subject:   "test",
		ActionUrl: "http://example.com",
		Body:      "test",
	}

	withPlaceholdersTemplate := settings.EmailTemplate{
		Subject:   "test",
		ActionUrl: "http://example.com" + settings.EmailPlaceholderToken,
		Body:      "test" + settings.EmailPlaceholderActionUrl,
	}

	scenarios := []struct {
		config      settings.MetaConfig
		expectError bool
	}{
		// zero values
		{
			settings.MetaConfig{},
			true,
		},
		// invalid data
		{
			settings.MetaConfig{
				AppName:                    strings.Repeat("a", 300),
				AppUrl:                     "test",
				SenderName:                 strings.Repeat("a", 300),
				SenderAddress:              "invalid_email",
				VerificationTemplate:       invalidTemplate,
				ResetPasswordTemplate:      invalidTemplate,
				ConfirmEmailChangeTemplate: invalidTemplate,
			},
			true,
		},
		// invalid data (missing required placeholders)
		{
			settings.MetaConfig{
				AppName:                    "test",
				AppUrl:                     "https://example.com",
				SenderName:                 "test",
				SenderAddress:              "test@example.com",
				VerificationTemplate:       noPlaceholdersTemplate,
				ResetPasswordTemplate:      noPlaceholdersTemplate,
				ConfirmEmailChangeTemplate: noPlaceholdersTemplate,
			},
			true,
		},
		// valid data
		{
			settings.MetaConfig{
				AppName:                    "test",
				AppUrl:                     "https://example.com",
				SenderName:                 "test",
				SenderAddress:              "test@example.com",
				VerificationTemplate:       withPlaceholdersTemplate,
				ResetPasswordTemplate:      withPlaceholdersTemplate,
				ConfirmEmailChangeTemplate: withPlaceholdersTemplate,
			},
			false,
		},
	}

	for i, scenario := range scenarios {
		result := scenario.config.Validate()

		if result != nil && !scenario.expectError {
			t.Errorf("(%d) Didn't expect error, got %v", i, result)
		}

		if result == nil && scenario.expectError {
			t.Errorf("(%d) Expected error, got nil", i)
		}
	}
}

func TestBackupsConfigValidate(t *testing.T) {
	scenarios := []struct {
		name           string
		config         settings.BackupsConfig
		expectedErrors []string
	}{
		{
			"zero value",
			settings.BackupsConfig{},
			[]string{},
		},
		{
			"invalid cron",
			settings.BackupsConfig{
				Cron:        "invalid",
				CronMaxKeep: 0,
			},
			[]string{"cron", "cronMaxKeep"},
		},
		{
			"invalid enabled S3",
			settings.BackupsConfig{
				S3: settings.S3Config{
					Enabled: true,
				},
			},
			[]string{"s3"},
		},
		{
			"valid data",
			settings.BackupsConfig{
				S3: settings.S3Config{
					Enabled:   true,
					Endpoint:  "example.com",
					Bucket:    "test",
					Region:    "test",
					AccessKey: "test",
					Secret:    "test",
				},
				Cron:        "*/10 * * * *",
				CronMaxKeep: 1,
			},
			[]string{},
		},
	}

	for _, s := range scenarios {
		result := s.config.Validate()

		// parse errors
		errs, ok := result.(validation.Errors)
		if !ok && result != nil {
			t.Errorf("[%s] Failed to parse errors %v", s.name, result)
			continue
		}

		// check errors
		if len(errs) > len(s.expectedErrors) {
			t.Errorf("[%s] Expected error keys %v, got %v", s.name, s.expectedErrors, errs)
		}
		for _, k := range s.expectedErrors {
			if _, ok := errs[k]; !ok {
				t.Errorf("[%s] Missing expected error key %q in %v", s.name, k, errs)
			}
		}
	}
}

func TestEmailTemplateValidate(t *testing.T) {
	scenarios := []struct {
		emailTemplate  settings.EmailTemplate
		expectedErrors []string
	}{
		// require values
		{
			settings.EmailTemplate{},
			[]string{"subject", "actionUrl", "body"},
		},
		// missing placeholders
		{
			settings.EmailTemplate{
				Subject:   "test",
				ActionUrl: "test",
				Body:      "test",
			},
			[]string{"actionUrl", "body"},
		},
		// valid data
		{
			settings.EmailTemplate{
				Subject:   "test",
				ActionUrl: "test" + settings.EmailPlaceholderToken,
				Body:      "test" + settings.EmailPlaceholderActionUrl,
			},
			[]string{},
		},
	}

	for i, s := range scenarios {
		result := s.emailTemplate.Validate()

		// parse errors
		errs, ok := result.(validation.Errors)
		if !ok && result != nil {
			t.Errorf("(%d) Failed to parse errors %v", i, result)
			continue
		}

		// check errors
		if len(errs) > len(s.expectedErrors) {
			t.Errorf("(%d) Expected error keys %v, got %v", i, s.expectedErrors, errs)
		}
		for _, k := range s.expectedErrors {
			if _, ok := errs[k]; !ok {
				t.Errorf("(%d) Missing expected error key %q in %v", i, k, errs)
			}
		}
	}
}

func TestEmailTemplateResolve(t *testing.T) {
	allPlaceholders := settings.EmailPlaceholderActionUrl + settings.EmailPlaceholderToken + settings.EmailPlaceholderAppName + settings.EmailPlaceholderAppUrl

	scenarios := []struct {
		emailTemplate     settings.EmailTemplate
		expectedSubject   string
		expectedBody      string
		expectedActionUrl string
	}{
		// no placeholders
		{
			emailTemplate: settings.EmailTemplate{
				Subject:   "subject:",
				Body:      "body:",
				ActionUrl: "/actionUrl////",
			},
			expectedSubject:   "subject:",
			expectedActionUrl: "/actionUrl/",
			expectedBody:      "body:",
		},
		// with placeholders
		{
			emailTemplate: settings.EmailTemplate{
				ActionUrl: "/actionUrl////" + allPlaceholders,
				Subject:   "subject:" + allPlaceholders,
				Body:      "body:" + allPlaceholders,
			},
			expectedActionUrl: fmt.Sprintf(
				"/actionUrl/%%7BACTION_URL%%7D%s%s%s",
				"token_test",
				"name_test",
				"url_test",
			),
			expectedSubject: fmt.Sprintf(
				"subject:%s%s%s%s",
				settings.EmailPlaceholderActionUrl,
				settings.EmailPlaceholderToken,
				"name_test",
				"url_test",
			),
			expectedBody: fmt.Sprintf(
				"body:%s%s%s%s",
				fmt.Sprintf(
					"/actionUrl/%%7BACTION_URL%%7D%s%s%s",
					"token_test",
					"name_test",
					"url_test",
				),
				"token_test",
				"name_test",
				"url_test",
			),
		},
	}

	for i, s := range scenarios {
		subject, body, actionUrl := s.emailTemplate.Resolve("name_test", "url_test", "token_test")

		if s.expectedSubject != subject {
			t.Errorf("(%d) Expected subject %q got %q", i, s.expectedSubject, subject)
		}

		if s.expectedBody != body {
			t.Errorf("(%d) Expected body \n%v got \n%v", i, s.expectedBody, body)
		}

		if s.expectedActionUrl != actionUrl {
			t.Errorf("(%d) Expected actionUrl \n%v got \n%v", i, s.expectedActionUrl, actionUrl)
		}
	}
}

func TestLogsConfigValidate(t *testing.T) {
	scenarios := []struct {
		config      settings.LogsConfig
		expectError bool
	}{
		// zero values
		{
			settings.LogsConfig{},
			false,
		},
		// invalid data
		{
			settings.LogsConfig{MaxDays: -10},
			true,
		},
		// valid data
		{
			settings.LogsConfig{MaxDays: 1},
			false,
		},
	}

	for i, scenario := range scenarios {
		result := scenario.config.Validate()

		if result != nil && !scenario.expectError {
			t.Errorf("(%d) Didn't expect error, got %v", i, result)
		}

		if result == nil && scenario.expectError {
			t.Errorf("(%d) Expected error, got nil", i)
		}
	}
}

func TestAuthProviderConfigValidate(t *testing.T) {
	scenarios := []struct {
		config      settings.AuthProviderConfig
		expectError bool
	}{
		// zero values (disabled)
		{
			settings.AuthProviderConfig{},
			false,
		},
		// zero values (enabled)
		{
			settings.AuthProviderConfig{Enabled: true},
			true,
		},
		// invalid data
		{
			settings.AuthProviderConfig{
				Enabled:      true,
				ClientId:     "",
				ClientSecret: "",
				AuthUrl:      "test",
				TokenUrl:     "test",
				UserApiUrl:   "test",
			},
			true,
		},
		// valid data (only the required)
		{
			settings.AuthProviderConfig{
				Enabled:      true,
				ClientId:     "test",
				ClientSecret: "test",
			},
			false,
		},
		// valid data (fill all fields)
		{
			settings.AuthProviderConfig{
				Enabled:      true,
				ClientId:     "test",
				ClientSecret: "test",
				AuthUrl:      "https://example.com",
				TokenUrl:     "https://example.com",
				UserApiUrl:   "https://example.com",
			},
			false,
		},
	}

	for i, scenario := range scenarios {
		result := scenario.config.Validate()

		if result != nil && !scenario.expectError {
			t.Errorf("(%d) Didn't expect error, got %v", i, result)
		}

		if result == nil && scenario.expectError {
			t.Errorf("(%d) Expected error, got nil", i)
		}
	}
}

func TestAuthProviderConfigSetupProvider(t *testing.T) {
	provider := auth.NewGithubProvider()

	// disabled config
	c1 := settings.AuthProviderConfig{Enabled: false}
	if err := c1.SetupProvider(provider); err == nil {
		t.Errorf("Expected error, got nil")
	}

	c2 := settings.AuthProviderConfig{
		Enabled:      true,
		ClientId:     "test_ClientId",
		ClientSecret: "test_ClientSecret",
		AuthUrl:      "test_AuthUrl",
		UserApiUrl:   "test_UserApiUrl",
		TokenUrl:     "test_TokenUrl",
	}
	if err := c2.SetupProvider(provider); err != nil {
		t.Error(err)
	}

	if provider.ClientId() != c2.ClientId {
		t.Fatalf("Expected ClientId %s, got %s", c2.ClientId, provider.ClientId())
	}

	if provider.ClientSecret() != c2.ClientSecret {
		t.Fatalf("Expected ClientSecret %s, got %s", c2.ClientSecret, provider.ClientSecret())
	}

	if provider.AuthUrl() != c2.AuthUrl {
		t.Fatalf("Expected AuthUrl %s, got %s", c2.AuthUrl, provider.AuthUrl())
	}

	if provider.UserApiUrl() != c2.UserApiUrl {
		t.Fatalf("Expected UserApiUrl %s, got %s", c2.UserApiUrl, provider.UserApiUrl())
	}

	if provider.TokenUrl() != c2.TokenUrl {
		t.Fatalf("Expected TokenUrl %s, got %s", c2.TokenUrl, provider.TokenUrl())
	}
}

func TestAgentEmbeddingModel(t *testing.T) {
	cfg := settings.AgentConfig{
		Embedding: settings.AgentEmbeddingConfig{
			Enabled:      true,
			DefaultModel: "embed-model",
			Providers: []settings.AgentEmbeddingProviderConfig{
				{
					Id:      "provider-a",
					Enabled: true,
					Models: []settings.AgentEmbeddingModel{
						{
							Name:            "embed",
							ProviderModelId: "embed-model",
							Enabled:         true,
						},
					},
				},
			},
		},
	}

	if got := cfg.EmbeddingModel(); got != "embed-model" {
		t.Fatalf("expected embed-model, got %q", got)
	}

	cfg.Embedding.DefaultModel = ""
	if got := cfg.EmbeddingModel(); got != "" {
		t.Fatalf("expected empty embedding model, got %q", got)
	}

	cfg.Embedding.DefaultModel = "embed-model"
	cfg.Embedding.Enabled = false
	if got := cfg.EmbeddingModel(); got != "" {
		t.Fatalf("expected disabled embedding model to be empty, got %q", got)
	}
}

func TestAgentEmbeddingProviderApiValidation(t *testing.T) {
	cases := []struct {
		name    string
		api     string
		wantErr bool
	}{
		{name: "openai", api: "openai-embeddings", wantErr: false},
		{name: "empty", api: "", wantErr: false},
		{name: "chat", api: "openai-chat", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := settings.AgentEmbeddingProviderConfig{
				Id:      "embed-main",
				Vendor:  "openai",
				Api:     tc.api,
				BaseUrl: "https://example.com/v1",
				Models: []settings.AgentEmbeddingModel{
					{Name: "embed", ProviderModelId: "embed", Enabled: true},
				},
			}
			err := cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
