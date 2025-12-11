package response


type ListPods struct {
	Name       string         `json:"name"`
	Namespace  string         `json:"namespace"`
	Containers []PodContainer `json:"containers"`
	CreatedAt  string         `json:"created_at"`
	StartedAt  string         `json:"started_at"`
	Status     string         `json:"status"`
	Conditions []PodCondition `json:"conditions"`
}


type PodCondition struct {
	Name               string `json:"name"`
	Status             string `json:"status"`
	LastTransitionTime string `json:"last_transition_time"`
	Reason             string `json:"reason,omitempty"`
}


type PodContainer struct {
	Name     string               `json:"name"`
	Statuses []PodContainerStatus `json:"statuses,omitempty"`
}


type PodContainerStatus struct {
	Name         string                    `json:"name"`
	Image        string                    `json:"image"`
	Started      bool                      `json:"started,omitempty"`
	Ready        bool                      `json:"ready,omitempty"`
	RestartCount int                       `json:"restart_count,omitempty"`
	States       []PodContainerStatusState `json:"states,omitempty"`
}


type PodContainerStatusState struct {
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
	Reason     string `json:"reason,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
}
