package orm

import (
	flow "github.com/Bilibotter/light-flow"
	"gorm.io/gorm"
	"strings"
	"time"
)

const (
	Begin int8 = iota
	Suspend
	Success
	Failure
)

const (
	dbHasCreate = "already exists"
)

type Persistence interface {
	InjectPersistence() error
}

type persistence struct {
	*gorm.DB
}

type Step struct {
	Id         string `gorm:"primaryKey;type:char(36)"`
	Name       string
	Status     int8
	ProcId     string `gorm:"type:char(36)"`
	FlowId     string `gorm:"type:char(36)"`
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}

type Process struct {
	Id         string `gorm:"primaryKey;type:char(36)"`
	Name       string
	Status     int8
	FlowId     string `gorm:"type:char(36)"`
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}

type Flow struct {
	Id         string `gorm:"primaryKey;type:char(36)"`
	Name       string
	Status     int8
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}

func NewPersistPlugin(db *gorm.DB) Persistence {
	p := &persistence{db}
	return p
}

func (p *persistence) CreateTables() error {
	if tables, err := p.Migrator().GetTables(); err == nil && len(tables) >= 3 {
		m := make(map[string]bool, len(tables))
		for _, t := range tables {
			m[t] = true
		}
		if m["flows"] && m["processes"] && m["steps"] {
			return nil
		}
	}
	if err := p.Migrator().CreateTable(&Flow{}); err != nil {
		// can't use errors.Is(xxx, err), so use strings.Contains instead
		if !strings.Contains(err.Error(), dbHasCreate) {
			return err
		}
	}
	if err := p.Migrator().CreateTable(&Process{}); err != nil {
		if !strings.Contains(err.Error(), dbHasCreate) {
			return err
		}
	}
	if err := p.Migrator().CreateTable(&Step{}); err != nil {
		if !strings.Contains(err.Error(), dbHasCreate) {
			return err
		}
	}
	return nil
}

func (p *persistence) InjectPersistence() error {
	if err := p.CreateTables(); err != nil {
		return err
	}
	flow.FlowPersist().OnInsert(p.InsertFlow).OnUpdate(p.UpdateFlow)
	flow.ProcPersist().OnInsert(p.InsertProc).OnUpdate(p.UpdateProc)
	flow.StepPersist().OnInsert(p.InsertStep).OnUpdate(p.UpdateStep)
	return nil
}

func (p *persistence) InsertFlow(wf flow.WorkFlow) error {
	foo := &Flow{
		Id:        wf.ID(),
		Name:      wf.Name(),
		Status:    Begin,
		CreatedAt: wf.StartTime(),
		UpdatedAt: wf.StartTime(),
	}
	return p.Create(foo).Error
}

func (p *persistence) UpdateFlow(wf flow.WorkFlow) error {
	now := time.Now()
	foo := &Flow{
		UpdatedAt: &now,
	}
	if wf.Success() {
		foo.Status = Success
	} else {
		foo.Status = Failure
	}
	if wf.Has(flow.Suspend) {
		foo.Status = Suspend
	}
	if wf.EndTime() != nil {
		foo.FinishedAt = wf.EndTime()
	}
	return p.Model(&Flow{}).Where("id = ?", wf.ID()).Updates(foo).Error
}

func (p *persistence) InsertProc(proc flow.Process) error {
	foo := &Process{
		Id:        proc.ID(),
		Name:      proc.Name(),
		Status:    Begin,
		FlowId:    proc.FlowID(),
		CreatedAt: proc.StartTime(),
		UpdatedAt: proc.StartTime(),
	}
	return p.Create(foo).Error
}

func (p *persistence) UpdateProc(proc flow.Process) error {
	now := time.Now()
	foo := &Process{
		UpdatedAt: &now,
	}
	if proc.Success() {
		foo.Status = Success
	} else {
		foo.Status = Failure
	}
	if proc.Has(flow.Suspend) {
		foo.Status = Suspend
	}
	if proc.EndTime() != nil {
		foo.FinishedAt = proc.EndTime()
	}
	return p.Model(&Process{}).Where("id = ?", proc.ID()).Updates(foo).Error
}

func (p *persistence) InsertStep(step flow.Step) error {
	foo := &Step{
		Id:        step.ID(),
		Name:      step.Name(),
		Status:    Begin,
		ProcId:    step.ProcessID(),
		FlowId:    step.FlowID(),
		CreatedAt: step.StartTime(),
		UpdatedAt: step.StartTime(),
	}
	return p.Create(foo).Error
}

func (p *persistence) UpdateStep(step flow.Step) error {
	now := time.Now()
	foo := &Step{
		UpdatedAt: &now,
	}
	if step.Success() {
		foo.Status = Success
	} else {
		foo.Status = Failure
	}
	if step.Has(flow.Suspend) {
		foo.Status = Suspend
	}
	if step.EndTime() != nil {
		foo.FinishedAt = step.EndTime()
	}
	return p.Model(&Step{}).Where("id = ?", step.ID()).Updates(foo).Error
}
