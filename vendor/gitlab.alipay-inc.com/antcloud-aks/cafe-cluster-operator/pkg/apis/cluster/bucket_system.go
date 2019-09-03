package cluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	systemBuckets = []*Bucket{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "system-apiserver-loopback",
			},
			Spec: BucketSpec{
				Weight:        10,
				ReservedQuota: 50,
				SharedQuota:   0,
				Priority:      SystemTopPriorityBand,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "system-controller-high",
			},
			Spec: BucketSpec{
				Weight:        10,
				ReservedQuota: 50,
				SharedQuota:   50,
				Priority:      SystemHighPriorityBand,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "system-scheduler",
			},
			Spec: BucketSpec{
				Weight:        10,
				ReservedQuota: 50,
				SharedQuota:   50,
				Priority:      SystemMediumPriorityBand,
			},
		},
	}
	systemBucketBindings = []*BucketBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "system-apiserver-loopback",
			},
			Spec: BucketBindingSpec{
				Rules: []*BucketBindingRule{
					{
						Field:  "user.name",
						Values: []string{"system:apiserver"},
					},
				},
				BucketRef: &BucketReference{
					Name: "system-apiserver-loopback",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "system-controller-high",
			},
			Spec: BucketBindingSpec{
				Rules: []*BucketBindingRule{
					{
						Field:  "user.name",
						Values: []string{"system:kube-controller-manager"},
					},
				},
				BucketRef: &BucketReference{
					Name: "system-controller-high",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "system-scheduler",
			},
			Spec: BucketBindingSpec{
				Rules: []*BucketBindingRule{
					{
						Field:  "user.name",
						Values: []string{"system:kube-scheduler"},
					},
				},
				BucketRef: &BucketReference{
					Name: "system-scheduler",
				},
			},
		},
	}
)
