package handlers

import (
	"sort"

	"common/utils"
	"connector-kube/src/dto/response"
	"connector-kube/src/kubernetes"

	"github.com/gofiber/fiber/v2"
)


func ListPods(ctx *fiber.Ctx) error {
	pods, err := kubernetes.Client.ListPods()

	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	result := []response.ListPods{}

	for _, pod := range pods {
		podData := response.ListPods{
			Name:      pod.ObjectMeta.Name,
			Namespace: pod.ObjectMeta.Namespace,
			CreatedAt: utils.TrimKubeTime(pod.ObjectMeta.CreationTimestamp),
			Status:    string(pod.Status.Phase),
		}

		containers := []response.PodContainer{}
		conditions := []response.PodCondition{}

		for _, container := range pod.Spec.Containers {
			containerData := response.PodContainer{
				Name: container.Name,
			}

			containers = append(containers, containerData)
		}

		for _, condition := range pod.Status.Conditions {
			conditionData := response.PodCondition{
				Name: string(condition.Type),
				Status: string(condition.Status),
				LastTransitionTime: utils.TrimKubeTime(condition.LastTransitionTime),
				Reason: condition.Reason,
			}

			conditions = append(conditions, conditionData)
		}

		sort.Slice(conditions, func(i, j int) bool {
			return conditions[i].LastTransitionTime > conditions[j].LastTransitionTime
		})

		if pod.Status.StartTime != nil {
			podData.StartedAt = utils.TrimKubeTime(*pod.Status.StartTime)
		}

		podData.Containers = containers
		podData.Conditions = conditions

		result = append(result, podData)
	}

	return ctx.JSON(result)
}
