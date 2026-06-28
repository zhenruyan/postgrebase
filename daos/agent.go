package daos

import (
	"github.com/zhenruyan/postgrebase/dbx"
	"github.com/zhenruyan/postgrebase/models"
)

// SaveAgentSession upserts an agent session.
func (dao *Dao) SaveAgentSession(session *models.AgentSession) error {
	return dao.Save(session)
}

// DeleteAgentSession removes an agent session and its messages/audit records.
func (dao *Dao) DeleteAgentSession(session *models.AgentSession) error {
	return dao.RunInTransaction(func(txDao *Dao) error {
		if _, err := txDao.NonconcurrentDB().
			Delete("_pb_agent_messages_", dbx.HashExp{"session_id": session.Id}).
			Execute(); err != nil {
			return err
		}
		if _, err := txDao.NonconcurrentDB().
			Delete("_pb_agent_audit_", dbx.HashExp{"session_id": session.Id}).
			Execute(); err != nil {
			return err
		}
		return txDao.Delete(session)
	})
}

// FindAgentSessionById returns a single agent session by id.
func (dao *Dao) FindAgentSessionById(id string) (*models.AgentSession, error) {
	session := &models.AgentSession{}
	if err := dao.ModelQuery(session).
		AndWhere(dbx.HashExp{"id": id}).
		Limit(1).
		One(session); err != nil {
		return nil, err
	}
	return session, nil
}

// FindAgentSessionsByProject returns sessions for a project, newest first.
// When project is empty, all sessions are returned.
func (dao *Dao) FindAgentSessionsByProject(project string) ([]*models.AgentSession, error) {
	sessions := []*models.AgentSession{}
	query := dao.ModelQuery(&models.AgentSession{})
	if project != "" {
		query = query.AndWhere(dbx.HashExp{"project_id": project})
	}
	if err := query.OrderBy("updated DESC").All(&sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// SaveAgentMessage persists a conversation item.
func (dao *Dao) SaveAgentMessage(message *models.AgentMessage) error {
	return dao.Save(message)
}

// FindAgentMessagesBySession returns the messages of a session, oldest first.
func (dao *Dao) FindAgentMessagesBySession(sessionID string) ([]*models.AgentMessage, error) {
	messages := []*models.AgentMessage{}
	if err := dao.ModelQuery(&models.AgentMessage{}).
		AndWhere(dbx.HashExp{"session_id": sessionID}).
		OrderBy("created ASC").
		All(&messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// SaveAgentAudit persists an audit record.
func (dao *Dao) SaveAgentAudit(record *models.AgentAuditRecord) error {
	return dao.Save(record)
}

// FindAgentAuditBySession returns the audit trail of a session, oldest first.
func (dao *Dao) FindAgentAuditBySession(sessionID string) ([]*models.AgentAuditRecord, error) {
	records := []*models.AgentAuditRecord{}
	if err := dao.ModelQuery(&models.AgentAuditRecord{}).
		AndWhere(dbx.HashExp{"session_id": sessionID}).
		OrderBy("created ASC").
		All(&records); err != nil {
		return nil, err
	}
	return records, nil
}

// FindAgentProjectConfig returns the persisted per-project agent config, if any.
func (dao *Dao) FindAgentProjectConfig(project string) (*models.AgentProjectConfig, error) {
	config := &models.AgentProjectConfig{}
	if err := dao.ModelQuery(config).
		AndWhere(dbx.HashExp{"project_id": project}).
		Limit(1).
		One(config); err != nil {
		return nil, err
	}
	return config, nil
}

// SaveAgentProjectConfig upserts a per-project agent config.
func (dao *Dao) SaveAgentProjectConfig(config *models.AgentProjectConfig) error {
	return dao.Save(config)
}
