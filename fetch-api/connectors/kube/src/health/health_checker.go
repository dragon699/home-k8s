package health

import (
	"time"

	"connector-kube/settings"
	"connector-kube/src/kubernetes"
	t "connector-kube/src/telemetry"

	"github.com/go-co-op/gocron"
)


type HealthChecker struct {
	Scheduler *gocron.Scheduler
}


func (instance *HealthChecker) CreateSchedule() {
	t.Log.Info("Scheduling health checks..")
	instance.updateStatus()

	if settings.Config.HealthJobID == nil {
		instance.updateSchedule()
	}
}


func (instance *HealthChecker) getNextInterval() int {
	if settings.Config.Healthy != nil && *settings.Config.Healthy {
		return settings.Config.HealthCheckIntervalSeconds
	} else {
		return settings.Config.HealthRetryIntervalSeconds
	}
}


func (instance *HealthChecker) getNextCheckTime() string {
	ts := time.Now().Add(
		time.Duration(
			instance.getNextInterval(),
		) * time.Second,
	)

	return ts.Format("2006-01-02T15:04:05")
}


func (instance *HealthChecker) updateStatus() {
	var wasHealthy bool
	var isHealthy bool

	wasHealthyPtr := settings.Config.Healthy

	if wasHealthyPtr != nil {
		wasHealthy = *wasHealthyPtr
	}

	lastCheckTime := time.Now().Format("2006-01-02T15:04:05")
	settings.Config.HealthLastCheck = &lastCheckTime
	healthStatus, healthStatusCode, err := kubernetes.Client.Ping()

	if err != nil {
		settings.Config.Healthy = nil

		if wasHealthyPtr != nil {
			instance.updateSchedule()
		} else {
			nextCheckTime := instance.getNextCheckTime()
			settings.Config.HealthNextCheck = &nextCheckTime
		}

		t.Log.Error("Health check failed, Kubernetes is unreachable", "error", err.Error())

		return
	}

	if healthStatus == "ok" {
		isHealthy = true
	} else {
		isHealthy = false
		settings.Config.Healthy = nil

		if wasHealthyPtr != nil {
			instance.updateSchedule()
		} else {
			nextCheckTime := instance.getNextCheckTime()
			settings.Config.HealthNextCheck = &nextCheckTime
		}

		t.Log.Warn("Health check failed, got unexpected response", "status", healthStatus)

		return
	}

	if healthStatusCode == 200 && isHealthy {
		isHealthy = true
		settings.Config.Healthy = &isHealthy

		if wasHealthyPtr != nil && wasHealthy {
			nextCheckTime := instance.getNextCheckTime()
			settings.Config.HealthNextCheck = &nextCheckTime
		} else {
			t.Log.Info("Health check successful")
			instance.updateSchedule()
		}
	} else {
		isHealthy = false
		settings.Config.Healthy = &isHealthy

		if wasHealthyPtr == nil || wasHealthy != isHealthy {
			instance.updateSchedule()
		} else {
			nextCheckTime := instance.getNextCheckTime()
			settings.Config.HealthNextCheck = &nextCheckTime
		}

		t.Log.Warn("Health check failed, Kubernetes is unhealthy", "status_code", healthStatusCode)
	}
}


func (instance *HealthChecker) updateSchedule() {
	if instance.Scheduler == nil {
		return
	}

	if settings.Config.HealthJobID != nil {
		instance.Scheduler.RemoveByTag(*settings.Config.HealthJobID)
	}

	interval := instance.getNextInterval()
	nextCheckTime := instance.getNextCheckTime()
	settings.Config.HealthNextCheck = &nextCheckTime

	jobTag := "health_check_kubernetes_api_server"
	job, _ := instance.Scheduler.Every(interval).Seconds().Do(instance.updateStatus)
	job.Tag(jobTag)
	settings.Config.HealthJobID = &jobTag
}
