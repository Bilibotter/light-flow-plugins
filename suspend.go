package light_flow_plugins

import (
	flow "github.com/Bilibotter/light-flow"
	"gorm.io/gorm"
	"strings"
	"time"
)

type SuspendPlugin interface {
	InjectSuspend() error
}

type Checkpoint struct {
	Id        string    `gorm:"column:id;primary_key"`
	Uid       string    `gorm:"column:uid"`
	Name      string    `gorm:"column:name;NOT NULL"`
	RecoverId string    `gorm:"column:recover_id"`
	ParentUid string    `gorm:"column:parent_uid"`
	RootUid   string    `gorm:"column:root_uid"`
	Scope     uint8     `gorm:"column:scope;NOT NULL"`
	Snapshot  []byte    `gorm:"column:snapshot"`
	CreatedAt time.Time `gorm:"type:datetime;column:created_at;"`
	UpdatedAt time.Time `gorm:"type:datetime;column:updated_at;"`
}

type RecoverRecord struct {
	RootUid   string    `gorm:"column:root_uid;NOT NULL"`
	RecoverId string    `gorm:"column:recover_id;primary_key"`
	Status    uint8     `gorm:"column:status;NOT NULL"`
	Name      string    `gorm:"column:name;NOT NULL"`
	CreatedAt time.Time `gorm:"type:datetime;column:created_at;"`
	UpdatedAt time.Time `gorm:"type:datetime;column:updated_at;"`
}

type suspendPlugin struct {
	*gorm.DB
}

func NewSuspendPlugin(db *gorm.DB) SuspendPlugin {
	return &suspendPlugin{
		DB: db,
	}
}

func (s *suspendPlugin) GetLatestRecord(rootUid string) (flow.RecoverRecord, error) {
	var record RecoverRecord
	result := s.
		Where("root_uid = ?", rootUid).
		Where("status = ?", flow.RecoverIdle).
		First(&record)
	if result.Error != nil {
		return &record, result.Error
	}
	return &record, nil
}

func (s *suspendPlugin) ListCheckpoints(recoverId string) ([]flow.CheckPoint, error) {
	var checkpoints []*Checkpoint
	result := s.
		Where("recover_id = ?", recoverId).
		Find(&checkpoints)
	if result.Error != nil {
		return nil, result.Error
	}
	cps := make([]flow.CheckPoint, len(checkpoints))
	for i, cp := range checkpoints {
		cps[i] = cp
	}
	return cps, nil
}

func (s *suspendPlugin) UpdateRecordStatus(record flow.RecoverRecord) error {
	result := s.Model(&RecoverRecord{}).
		Where("recover_id = ?", record.GetRecoverId()).
		Update("status", record.GetStatus())
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *suspendPlugin) SaveCheckpointAndRecord(checkpoints []flow.CheckPoint, record flow.RecoverRecord) error {
	tx := s.Begin()
	cps := make([]*Checkpoint, len(checkpoints))
	for i, cp := range checkpoints {
		checkpoint := &Checkpoint{
			Id:        cp.GetId(),
			Uid:       cp.GetUid(),
			Name:      cp.GetName(),
			Snapshot:  cp.GetSnapshot(),
			RecoverId: cp.GetRecoverId(),
			ParentUid: cp.GetParentUid(),
			RootUid:   cp.GetRootUid(),
			Scope:     cp.GetScope(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		cps[i] = checkpoint
	}
	if err := tx.Create(&cps).Error; err != nil {
		tx.Rollback()
		panic(err)
	}
	rcd := &RecoverRecord{
		RootUid:   record.GetRootUid(),
		RecoverId: record.GetRecoverId(),
		Status:    record.GetStatus(),
		Name:      record.GetName(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := tx.Create(rcd).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

func (s *suspendPlugin) InjectSuspend() error {
	flow.SuspendPersist(s)
	return s.CreateTables()
}

func (s *suspendPlugin) CreateTables() error {
	if tables, err := s.Migrator().GetTables(); err == nil && len(tables) >= 3 {
		m := make(map[string]bool, len(tables))
		for _, t := range tables {
			m[t] = true
		}
		if m["recover_records"] && m["checkpoints"] {
			return nil
		}
	}
	if err := s.Migrator().CreateTable(&RecoverRecord{}); err != nil {
		// can't use errors.Is(xxx, err), so use strings.Contains instead
		if !strings.Contains(err.Error(), dbHasCreate) {
			return err
		}
	}
	if err := s.Migrator().CreateTable(&Checkpoint{}); err != nil {
		if !strings.Contains(err.Error(), dbHasCreate) {
			return err
		}
	}
	return nil
}

func (c *Checkpoint) GetId() string {
	return c.Id
}

func (c *Checkpoint) GetUid() string {
	return c.Uid
}

func (c *Checkpoint) GetName() string {
	return c.Name
}

func (c *Checkpoint) GetParentUid() string {
	return c.ParentUid
}

func (c *Checkpoint) GetRootUid() string {
	return c.RootUid
}

func (c *Checkpoint) GetScope() uint8 {
	return c.Scope
}

func (c *Checkpoint) GetRecoverId() string {
	return c.RecoverId
}

func (c *Checkpoint) GetSnapshot() []byte {
	return c.Snapshot
}

func (r *RecoverRecord) GetRootUid() string {
	return r.RootUid
}

func (r *RecoverRecord) GetRecoverId() string {
	return r.RecoverId
}

func (r *RecoverRecord) GetStatus() uint8 {
	return r.Status
}

func (r *RecoverRecord) GetName() string {
	return r.Name
}
