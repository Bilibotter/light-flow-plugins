# Suspend插件开发文档

## 使用Suspend插件

###  获取Suspend插件

要使用LightFlow框架提供的ORM插件，请运行以下命令：

```go
go get github.com/Bilibotter/light-flow-plugins/orm
```

### 使用前准备

在使用插件之前，请确保数据库中`recover_records`和`checkpoints`这两张表未被其他业务占用，以避免数据冲突。

### 设置数据库连接并注入插件

在代码中设置数据库连接并注入持久化插件，示例如下：

```go
db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
if err != nil {
    log.Fatalf("failed to connect to database: %v", err)
}

err = plugins.NewPersistPlugin(db).InjectPersistence()
if err != nil {
    log.Fatalf("failed to inject persistence plugin: %v", err)
}
```

## 自定义挂起插件实现

### 概述

在LightFlow框架中，用户可以实现自定义的持久化插件，以便在任务恢复过程中保存和恢复上下文数据。此插件需要实现两个核心接口：`RecoverRecord` 和 `Checkpoint`，并提供相应的结构体定义和数据库表。

###  结构体定义

您需要定义以下结构体，以便与数据库进行交互：

```go
type Checkpoint struct {
	Id        string    `gorm:"column:id;primary_key"`
	Uid       string    `gorm:"column:uid"` // current unit ID
	Name      string    `gorm:"column:name;NOT NULL"`
	RecoverId string    `gorm:"column:recover_id"`
	ParentUid string    `gorm:"column:parent_uid"`
	RootUid   string    `gorm:"column:root_uid"` // Equal to Flow_Id
	Scope     uint8     `gorm:"column:scope;NOT NULL"` // Equal to flow.StepScope or flow.ProcessScope or flow.FlowScope
	Snapshot  []byte    `gorm:"column:snapshot"`
	CreatedAt time.Time `gorm:"type:datetime;column:created_at;"`
	UpdatedAt time.Time `gorm:"type:datetime;column:updated_at;"`
}
```

### 需实现的方法

#### 1. 定义插件

```go
type suspendPlugin struct {
	*gorm.DB
}
```

#### 2. 获取最新记录

实现 `GetLatestRecord` 方法以获取最新的恢复记录。

```go
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
```

#### 3. 列出检查点

实现 `ListCheckpoints` 方法以列出与给定恢复ID相关的检查点。

```go
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
```

#### 4. 更新记录状态

实现 `UpdateRecordStatus` 方法以更新恢复记录的状态。

```go
func (s *suspendPlugin) UpdateRecordStatus(record flow.RecoverRecord) error {
	result := s.Model(&RecoverRecord{}).
		Where("recover_id = ?", record.GetRecoverId()).
		Update("status", record.GetStatus())
	if result.Error != nil {
		return result.Error
	}
	return nil
}
```

#### 5. 保存检查点和记录

实现 `SaveCheckpointAndRecord` 方法以同时保存检查点和恢复记录。

```go
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
```

### 设置插件

```go
import (
    "github.com/Bilibotter/light-flow/flow"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

func init() {
    db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    flow.SuspendPersist(&suspendPlugin{db})
}
```

