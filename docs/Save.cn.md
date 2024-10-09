# 持久化插件开发文档

## 使用orm插件

要使用LightFlow框架提供的ORM插件，请运行以下命令：

```go
go get github.com/Bilibotter/light-flow-plugins/orm
```

### 使用前准备

在使用插件之前，请确保数据库中`steps`、`processes`、`flows`这三张表未被其他业务占用，以避免数据冲突。

### 设置数据库连接并注入插件

```go
import (
	plugins "github.com/Bilibotter/light-flow-plugins/orm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
)

func init() {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	err = plugins.NewPersistPlugin(db).InjectPersistence()
	if err != nil {
		log.Fatalf("failed to inject persistence plugin: %v", err)
	}
}
```

------

## 自定义持久化插件编写指南

在任务执行过程中，持久化插件用于保存 `Flow`、`Process` 和 `Step` 的执行记录和状态。以下是指导用户如何编写自定义持久化插件的详细文档。

### 1. 定义数据结构

首先，您需要定义持久化的数据结构。您可以在现有的结构体中添加任意字段，以满足特定需求。以下是基本的结构体定义：

```go
type Step struct {
	Id         string     `gorm:"primaryKey;type:char(36)"`
	Name       string
	Status     int8
	ProcId     string     `gorm:"type:char(36)"`
	FlowId     string     `gorm:"type:char(36)"`
	AppId      string     // 新增字段
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}

type Process struct {
	Id         string     `gorm:"primaryKey;type:char(36)"`
	Name       string
	Status     int8
	FlowId     string     `gorm:"type:char(36)"`
	AppId      string     // 新增字段
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}

type Flow struct {
	Id         string     `gorm:"primaryKey;type:char(36)"`
	Name       string
	Status     int8
	AppId      string     // 新增字段
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}
```

------

### 2. 从上下文中获取值

为了从上下文中获取新添加的字段（例如 `AppId`），您需要在插件的实现方法中进行相应的修改。可以通过 `Context` 来访问这些值。

#### 示例：获取 `AppId`

在插入或更新操作中，您可以通过以下方式从上下文中获取 `AppId`：

```go
func (p *persistence) InsertFlow(wf flow.WorkFlow, ctx context.Context) error {
	appId, _ := wf.Get("appId") // 从上下文获取 AppId

	foo := &Flow{
		Id:        wf.ID(),
		Name:      wf.Name(),
		Status:    Begin,
		AppId:     appId.(string), // 设置 AppId
		CreatedAt: wf.StartTime(),
		UpdatedAt: wf.StartTime(),
	}
	return p.Create(foo).Error
}
```

------

### 3. 实现持久化方法

在实现持久化方法时，确保调用上下文中的值并将其注入到数据结构中。例如，在更新方法中也要添加相应的逻辑：

```go
func (p *persistence) UpdateFlow(wf flow.WorkFlow, ctx context.Context) error {
	now := time.Now()
	appId, _ := wf.Get("appId")

	foo := &Flow{
        AppId: appId.(string), // 设置 AppId
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

### 4. 注册持久化插件

在注册持久化插件时，确保将新方法与相应的插入和更新操作关联：

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

