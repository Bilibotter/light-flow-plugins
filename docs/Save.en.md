# Persistence Plugin Development Documentation

## Using the ORM Plugin

To use the ORM plugin provided by the LightFlow framework, run the following command:

```go
go get github.com/Bilibotter/light-flow-plugins/orm
```

### Preparation Before Use

Before using the plugin, ensure that the tables `steps`, `processes`, and `flows` in your database are not occupied by other business processes to avoid data conflicts.

### Setting Up Database Connection and Injecting the Plugin

```go
db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
if err != nil {
    t.Fatalf("Failed to open database: %v", err)
}
err = plugins.NewPersistPlugin(db).InjectPersistence()
if err != nil {
    t.Logf("Error injecting persistence: %v", err)
    return
}
```

------

## Guide to Writing Custom Persistence Plugins

During task execution, persistence plugins are used to save the execution records and states of `Flow`, `Process`, and `Step`. Below is a detailed guide on how to write custom persistence plugins.

### 1. Define Data Structures

First, you need to define the data structures for persistence. You can add any fields to existing structs to meet specific needs. Here are the basic struct definitions:

```go
type Step struct {
	Id         string     `gorm:"primaryKey;type:char(36)"`
	Name       string
	Status     int8
	ProcId     string     `gorm:"type:char(36)"`
	FlowId     string     `gorm:"type:char(36)"`
	AppId      string     // New field
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}

type Process struct {
	Id         string     `gorm:"primaryKey;type:char(36)"`
	Name       string
	Status     int8
	FlowId     string     `gorm:"type:char(36)"`
	AppId      string     // New field
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}

type Flow struct {
	Id         string     `gorm:"primaryKey;type:char(36)"`
	Name       string
	Status     int8
	AppId      string     // New field
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}
```

------

### 2. Retrieve Values from Context

To retrieve newly added fields (like `AppId`) from the context, you need to modify your implementation methods in the plugin accordingly. You can access these values through `Context`.

#### Example: Getting `AppId`

In insert or update operations, you can retrieve `AppId` from the context as follows:

```go
func (p *persistence) InsertFlow(wf flow.WorkFlow, ctx context.Context) error {
	appId, _ := wf.Get("appId") // Get AppId from context

	foo := &Flow{
		Id:        wf.ID(),
		Name:      wf.Name(),
		Status:    Begin,
		AppId:     appId.(string), // Set AppId
		CreatedAt: wf.StartTime(),
		UpdatedAt: wf.StartTime(),
	}
	return p.Create(foo).Error
}
```

------

### 3. Implement Persistence Methods

When implementing persistence methods, ensure that you call values from the context and inject them into your data structures. For example, add corresponding logic in update methods as well:

```go
func (p *persistence) UpdateFlow(wf flow.WorkFlow, ctx context.Context) error {
	now := time.Now()
	appId, _ := wf.Get("appId")

	foo := &Flow{
        AppId: appId.(string), // Set AppId
		UpdatedAt: &now,
	}
	if wf.Success() {
		foo.Status = Success
	} else {
		foo.Status = Failure
	}
	if wf.Has(flow.Suspend) {
		foo.Status = Suspend;
	}
	if wf.EndTime() != nil {
		foo.FinishedAt = wf.EndTime();
	}
	return p.Model(&Flow{}).Where("id = ?", wf.ID()).Updates(foo).Error;
}
```

------

### 4. Registering Persistence Plugins

When registering persistence plugins, ensure that new methods are associated with the corresponding insert and update operations:

```go
func (p *persistence) InjectPersistence() {
	flow.FlowPersist().
        OnInsert(p.InsertFlow).OnUpdate(p.UpdateFlow)
	flow.ProcPersist().
        OnInsert(p.InsertProc).OnUpdate(p.UpdateProc)
	flow.StepPersist().
        OnInsert(p.InsertStep).OnUpdate(p.UpdateStep)
}
```