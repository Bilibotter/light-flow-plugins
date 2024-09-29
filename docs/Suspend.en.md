# Suspend Plugin Development Documentation

## Using the Suspend Plugin

### Getting the Suspend Plugin

To use the ORM plugin provided by the LightFlow framework, run the following command:

```go
go get github.com/Bilibotter/light-flow-plugins/orm
```

### Preparation Before Use

Before using the plugin, ensure that the `recover_records` and `checkpoints` tables in the database are not occupied by other business processes to avoid data conflicts.

### Setting Up Database Connection and Injecting the Plugin

In your code, set up the database connection and inject the persistence plugin as shown below:

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

## Custom Suspend Plugin Implementation

### Overview

In the LightFlow framework, users can implement custom persistence plugins to save and restore context data during task recovery. This plugin needs to implement two core interfaces: `RecoverRecord` and `Checkpoint`, along with the corresponding struct definitions and database tables.

### Struct Definitions

You need to define the following structs to interact with the database:

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

### Methods to Implement

#### 1. Define the Plugin

```go
type suspendPlugin struct {
    *gorm.DB
}
```

#### 2. Get the Latest Record

Implement the `GetLatestRecord` method to retrieve the latest recovery record.

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

#### 3. List Checkpoints

Implement the `ListCheckpoints` method to list checkpoints associated with a given recovery ID.

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

#### 4. Update Record Status

Implement the `UpdateRecordStatus` method to update the status of the recovery record.

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

#### 5. Save Checkpoints and Records

Implement the `SaveCheckpointAndRecord` method to save both checkpoints and recovery records.

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

### Setting Up the Plugin

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

