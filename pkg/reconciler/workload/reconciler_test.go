/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package workload

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	workloadfake "github.com/crossplane/addon-oam-kubernetes-remote/pkg/reconciler/workload/fake"
)

var _ reconcile.Reconciler = &Reconciler{}

func TestReconciler(t *testing.T) {
	type args struct {
		m manager.Manager
		w Kind
		o []ReconcilerOption
	}

	type want struct {
		result reconcile.Result
		err    error
	}

	errBoom := errors.New("boom")

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"GetWorkloadError": {
			reason: "Any error (except not found) encountered while getting the resource under reconciliation should be returned.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(errBoom)},
					Scheme: fake.SchemeWith(&workloadfake.Workload{}),
				},
				w: Kind(fake.GVK(&workloadfake.Workload{})),
			},
			want: want{err: errors.Wrap(errBoom, errGetWorkload)},
		},
		"WorkloadNotFound": {
			reason: "Not found errors encountered while getting the resource under reconciliation should be ignored.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(kerrors.NewNotFound(schema.GroupResource{}, ""))},
					Scheme: fake.SchemeWith(&workloadfake.Workload{}),
				},
				w: Kind(fake.GVK(&workloadfake.Workload{})),
			},
			want: want{result: reconcile.Result{}},
		},
		"TranslateWorkloadError": {
			reason: "Failure to translate Workload into KubernetesApplication should be returned.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil),
						MockStatusUpdate: func(_ context.Context, obj runtime.Object, _ ...client.UpdateOption) error {
							got := obj.(Workload)

							if diff := cmp.Diff(v1alpha1.ReasonReconcileError, got.GetCondition(v1alpha1.TypeSynced).Reason); diff != "" {
								return errors.Errorf("MockStatusUpdate: -want, +got: %s", diff)
							}

							if diff := cmp.Diff(errors.Wrap(errBoom, errTranslateWorkload).Error(), got.GetCondition(v1alpha1.TypeSynced).Message); diff != "" {
								return errors.Errorf("MockStatusUpdate: -want, +got: %s", diff)
							}

							return nil
						},
					},
					Scheme: fake.SchemeWith(&workloadfake.Workload{}),
				},
				w: Kind(fake.GVK(&workloadfake.Workload{})),
				o: []ReconcilerOption{WithTranslator(TranslateFn(func(_ context.Context, _ Workload) ([]Object, error) {
					return nil, errBoom
				}))},
			},
			want: want{result: reconcile.Result{RequeueAfter: shortWait}},
		},
		"WrapWorkloadError": {
			reason: "Failure to translate Workload into KubernetesApplication should be returned.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil),
						MockStatusUpdate: func(_ context.Context, obj runtime.Object, _ ...client.UpdateOption) error {
							got := obj.(Workload)

							if diff := cmp.Diff(v1alpha1.ReasonReconcileError, got.GetCondition(v1alpha1.TypeSynced).Reason); diff != "" {
								return errors.Errorf("MockStatusUpdate: -want, +got: %s", diff)
							}

							if diff := cmp.Diff(errors.Wrap(errBoom, errTranslateWorkload).Error(), got.GetCondition(v1alpha1.TypeSynced).Message); diff != "" {
								return errors.Errorf("MockStatusUpdate: -want, +got: %s", diff)
							}

							return nil
						},
					},
					Scheme: fake.SchemeWith(&workloadfake.Workload{}),
				},
				w: Kind(fake.GVK(&workloadfake.Workload{})),
				o: []ReconcilerOption{
					WithTranslator(NewObjectTranslatorWithWrappers(func(_ context.Context, _ Workload) ([]Object, error) {
						return nil, nil
					}, func(ctx context.Context, w Workload, obj []Object) ([]Object, error) {
						return nil, errBoom
					})),
				},
			},
			want: want{result: reconcile.Result{RequeueAfter: shortWait}},
		},
		"ApplyError": {
			reason: "Failure to apply Workload translate should be returned.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil),
						MockStatusUpdate: func(_ context.Context, obj runtime.Object, _ ...client.UpdateOption) error {
							got := obj.(Workload)

							if diff := cmp.Diff(v1alpha1.ReasonReconcileError, got.GetCondition(v1alpha1.TypeSynced).Reason); diff != "" {
								return errors.Errorf("MockStatusUpdate: -want, +got: %s", diff)
							}

							if diff := cmp.Diff(errors.Wrap(errBoom, errApplyWorkloadTranslation).Error(), got.GetCondition(v1alpha1.TypeSynced).Message); diff != "" {
								return errors.Errorf("MockStatusUpdate: -want, +got: %s", diff)
							}

							return nil
						},
					},
					Scheme: fake.SchemeWith(&workloadfake.Workload{}),
				},
				w: Kind(fake.GVK(&workloadfake.Workload{})),
				o: []ReconcilerOption{
					WithTranslator(TranslateFn(func(ctx context.Context, w Workload) ([]Object, error) {
						return []Object{
							&appsv1.Deployment{},
							&appsv1.Deployment{},
						}, nil
					})),
					WithApplicator(resource.ApplyFn(func(_ context.Context, _ client.Client, _ runtime.Object, _ ...resource.ApplyOption) error {
						return errBoom
					}))},
			},
			want: want{result: reconcile.Result{RequeueAfter: shortWait}},
		},
		"Successful": {
			reason: "Successful reconciliaton should result in requeue after long wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet:          test.NewMockGetFn(nil),
						MockStatusUpdate: test.NewMockStatusUpdateFn(nil),
					},
					Scheme: fake.SchemeWith(&workloadfake.Workload{}),
				},
				w: Kind(fake.GVK(&workloadfake.Workload{})),
				o: []ReconcilerOption{WithApplicator(resource.ApplyFn(func(_ context.Context, _ client.Client, _ runtime.Object, _ ...resource.ApplyOption) error {
					return nil
				}))},
			},
			want: want{result: reconcile.Result{RequeueAfter: longWait}},
		},
		"SuccessfulStatusUpdateError": {
			reason: "Successful reconciliaton should result in requeue after long wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet:          test.NewMockGetFn(nil),
						MockStatusUpdate: test.NewMockStatusUpdateFn(errBoom),
					},
					Scheme: fake.SchemeWith(&workloadfake.Workload{}),
				},
				w: Kind(fake.GVK(&workloadfake.Workload{})),
				o: []ReconcilerOption{WithApplicator(resource.ApplyFn(func(_ context.Context, _ client.Client, _ runtime.Object, _ ...resource.ApplyOption) error {
					return nil
				}))},
			},
			want: want{
				result: reconcile.Result{RequeueAfter: longWait},
				err:    errors.Wrap(errBoom, errUpdateWorkloadStatus),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewReconciler(tc.args.m, tc.args.w, tc.args.o...)
			got, err := r.Reconcile(reconcile.Request{})

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\nReason: %s\nr.Reconcile(...): -want error, +got error:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.result, got); diff != "" {
				t.Errorf("\nReason: %s\nr.Reconcile(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}
