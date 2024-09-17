package test

import (
	"bufio"
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	plugins "github.com/Bilibotter/light-flow-plugins"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

var db *gorm.DB
var dsn string

func stringStatus(status int8) string {
	switch status {
	case plugins.Begin:
		return "Begin"
	case plugins.Suspend:
		return "Suspend"
	case plugins.Success:
		return "Success"
	case plugins.Failure:
		return "Failure"
	default:
		return "Unknown"
	}
}

func readDBConfig(filePath string) (username, password, host, dbname string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", "", "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	config := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return "", "", "", "", fmt.Errorf("invalid config line: %s", line)
		}
		config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	if err = scanner.Err(); err != nil {
		return "", "", "", "", err
	}
	username = config["username"]
	password = config["password"]
	host = config["host"]
	dbname = config["dbname"]
	return
}

func TestMain(m *testing.M) {
	username, password, host, dbname, err := readDBConfig("mysql.txt")
	if err != nil {
		log.Fatalf("Failureed to read DB config: %v", err)
	}
	dataSourceName := "%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local"
	dataSourceName = fmt.Sprintf(dataSourceName, username, password, host, dbname)
	dsn = dataSourceName
	for i := 0; i < 3; i++ {
		db, err = gorm.Open(mysql.Open(dataSourceName), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failureed to open database: %v", err)
		}
	}
	os.Exit(m.Run())
}

func CheckFlowPersist(t *testing.T, ff flow.FinishedWorkFlow, expect int) {
	t.Logf("Checking Flow %s", ff.Name())
	var f plugins.Flow
	count := 0
	if err := db.Where("id = ?", ff.ID()).First(&f).Error; err != nil {
		t.Errorf("Error getting Flow %s: %s", ff.Name(), err.Error())
	} else {
		count += 1
	}
	if ff.Name() != f.Name {
		t.Errorf("Flow %s has wrong name: %s", ff.Name(), f.Name)
	}
	if ff.ID() != f.Id {
		t.Errorf("Flow %s has wrong ID: %s", ff.Name(), f.Id)
	}
	if ff.StartTime().Sub(*f.CreatedAt) > time.Second {
		t.Errorf("Flow %s has wrong start time: %s, expected %s", ff.Name(), f.CreatedAt.UTC(), ff.StartTime().UTC())
	}
	if ff.EndTime().Sub(*f.UpdatedAt) > time.Second {
		t.Errorf("Flow %s has wrong end time: %s, expected %s", ff.Name(), f.UpdatedAt.UTC(), ff.EndTime().UTC())
	}
	if ff.EndTime().Sub(*f.FinishedAt) > time.Second {
		t.Errorf("Flow %s has wrong end time: %s, expected %s", ff.Name(), f.FinishedAt.UTC(), ff.EndTime().UTC())
	}
	if ff.Success() && f.Status != plugins.Success {
		t.Errorf("Flow %s should be Success but is %s", ff.Name(), stringStatus(f.Status))
	} else if !ff.Success() && f.Status != plugins.Failure {
		t.Errorf("Flow %s should be Failure but is %s", ff.Name(), stringStatus(f.Status))
	}
	if ff.Success() && f.Status != plugins.Success {
		t.Errorf("Flow %s should be Success but is %s", ff.Name(), stringStatus(f.Status))
	} else if !ff.Success() && f.Status != plugins.Failure {
		t.Errorf("Flow %s should be Failure but is %s", ff.Name(), stringStatus(f.Status))
	}
	t.Logf("Check Flow[ %s ] complete", ff.Name())
	for _, proc := range ff.Processes() {
		if !proc.Has(flow.Pending) {
			continue
		}
		var p plugins.Process
		if err := db.Where("id = ?", proc.ID()).First(&p).Error; err != nil {
			t.Errorf("Error getting Process %s: %s", proc.Name(), err.Error())
		} else {
			count += 1
		}
		if proc.Name() != p.Name {
			t.Errorf("Process %s has wrong name: %s", proc.Name(), p.Name)
		}
		if proc.ID() != p.Id {
			t.Errorf("Process %s has wrong ID: %s", proc.Name(), p.Id)
		}
		if proc.StartTime().Sub(*p.CreatedAt) > time.Second {
			t.Errorf("Process %s has wrong start time: %s, expected %s", proc.Name(), p.CreatedAt.UTC(), proc.StartTime().UTC())
		}
		if proc.EndTime().Sub(*p.UpdatedAt) > time.Second {
			t.Errorf("Process %s has wrong end time: %s, expected %s", proc.Name(), p.UpdatedAt.UTC(), proc.EndTime().UTC())
		}
		if proc.EndTime().Sub(*p.FinishedAt) > time.Second {
			t.Errorf("Process %s has wrong end time: %s, expected %s", proc.Name(), p.FinishedAt.UTC(), proc.EndTime().UTC())
		}
		if proc.Success() && p.Status != plugins.Success {
			t.Errorf("Process %s should be Success but is %s", proc.Name(), stringStatus(p.Status))
		} else if !proc.Success() && p.Status != plugins.Failure {
			t.Errorf("Process %s should be Failure but is %s", proc.Name(), stringStatus(p.Status))
		}
		t.Logf("Check [Process: %s ] complete", proc.Name())
		for _, step := range proc.Steps() {
			if !step.Has(flow.Pending) {
				continue
			}
			var s plugins.Step
			if err := db.Where("id = ?", step.ID()).First(&s).Error; err != nil {
				t.Errorf("Error getting Step %s: %s", step.Name(), err.Error())
			} else {
				count += 1
			}
			if step.Name() != s.Name {
				t.Errorf("Step %s has wrong name: %s", step.Name(), s.Name)
			}
			if step.ID() != s.Id {
				t.Errorf("Step %s has wrong ID: %s", step.Name(), s.Id)
			}
			if step.StartTime().Sub(*s.CreatedAt) > time.Second {
				t.Errorf("Step %s has wrong start time: %s, expected %s", step.Name(), s.CreatedAt.UTC(), step.StartTime().UTC())
			}
			if step.EndTime().Sub(*s.UpdatedAt) > time.Second {
				t.Errorf("Step %s has wrong end time: %s, expected %s", step.Name(), s.UpdatedAt.UTC(), step.EndTime().UTC())
			}
			if step.EndTime() == nil {
				t.Errorf("Step %s has no end time", step.Name())
			}
			if s.FinishedAt == nil {
				if s.CreatedAt != nil {
					t.Errorf("Step %s has no finished at time, but has created at time", step.Name())
				} else {
					t.Errorf("Step %s has no finished at time", step.Name())
				}
			}
			if step.EndTime().Sub(*s.FinishedAt) > time.Second {
				t.Errorf("Step %s has wrong end time: %s, expected %s", step.Name(), s.FinishedAt.UTC(), step.EndTime().UTC())
			}
			if step.Success() && s.Status != plugins.Success {
				t.Errorf("Step %s should be Success but is %s", step.Name(), stringStatus(s.Status))
			} else if !step.Success() && s.Status != plugins.Failure {
				t.Errorf("Step %s should be Failure but is %s", step.Name(), stringStatus(s.Status))
			}
			t.Logf("Check [Step: %s ] complete", step.Name())
		}
	}
	if count != expect {
		t.Errorf("Expected %d entities, got %d", expect, count)
	}
}

func TestMultipleTimesPersist(t *testing.T) {
	db0, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failureed to open database: %v", err)
	}
	if err = plugins.NewPersistPlugin(db0).InjectPersistence(); err != nil {
		t.Logf("Error injecting persistence: %v", err)
		return
	}
	wf := flow.RegisterFlow("TestMultipleTimesPersist")
	proc := wf.Process("TestMultipleTimesPersist")
	proc.NameStep(func(_ flow.Step) (any, error) {
		return "hello", nil
	}, "1")
	for i := 0; i < 8; i++ {
		ff := flow.DoneFlow("TestMultipleTimesPersist", nil)
		CheckFlowPersist(t, ff, 3)
	}
}

func TestSuccessStepPersist(t *testing.T) {
	db0, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failureed to open database: %v", err)
	}
	if err = plugins.NewPersistPlugin(db0).InjectPersistence(); err != nil {
		t.Logf("Error injecting persistence: %v", err)
		return
	}
	wf := flow.RegisterFlow("TestSuccessStepPersist")
	proc := wf.Process("TestSuccessStepPersist")
	proc.NameStep(func(_ flow.Step) (any, error) {
		return "hello", nil
	}, "1")
	ff := flow.DoneFlow("TestSuccessStepPersist", nil)
	CheckFlowPersist(t, ff, 3)
}

func TestFailureStepPersist(t *testing.T) {
	db0, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failureed to open database: %v", err)
	}
	if err = plugins.NewPersistPlugin(db0).InjectPersistence(); err != nil {
		t.Logf("Error injecting persistence: %v", err)
		return
	}
	wf := flow.RegisterFlow("TestFailureStepPersist")
	proc := wf.Process("TestFailureStepPersist")
	proc.NameStep(func(_ flow.Step) (any, error) {
		return nil, fmt.Errorf("failure")
	}, "1")
	ff := flow.DoneFlow("TestFailureStepPersist", nil)
	CheckFlowPersist(t, ff, 3)
}
