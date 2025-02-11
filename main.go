package main

import (
	"context"
	"flag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	argoApi "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	projectApi "github.com/openshift/api/project"
	corev1 "k8s.io/api/core/v1"
	k8sApi "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	cdPipeApiV1 "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/cdpipeline"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/clustersecret"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/argocd"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/aws"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/objectmodifier"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/webhook"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	buildInfo "github.com/epam/edp-common/pkg/config"
	edpCompApi "github.com/epam/edp-component-operator/api/v1"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	cdPipelineOperatorLock = "edp-cd-pipeline-operator-lock"
	ctrlManagerDefaultPort = 9443
)

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", cluster.RunningInCluster(),
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	mode, err := cluster.GetDebugMode()
	if err != nil {
		setupLog.Error(err, "unable to get debug mode value")
		os.Exit(1)
	}

	opts := zap.Options{
		Development: mode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(cdPipeApiV1.AddToScheme(scheme))
	utilruntime.Must(codebaseApi.AddToScheme(scheme))
	utilruntime.Must(edpCompApi.AddToScheme(scheme))
	utilruntime.Must(k8sApi.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(projectApi.Install(scheme))
	utilruntime.Must(argoApi.AddToScheme(scheme))

	v := buildInfo.Get()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("Starting the CD Pipeline Operator",
		"version", v.Version,
		"git-commit", v.GitCommit,
		"git-tag", v.GitTag,
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-client", v.KubectlVersion,
		"platform", v.Platform,
	)

	ns, err := cluster.GetWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get watch namespace")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		Port:                   ctrlManagerDefaultPort,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       cdPipelineOperatorLock,
		Namespace:              ns,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// this operator requires access to resources outside his namespace
	// for example: create a RoleBinding for each stage of each codebase (which are located in separate namespaces)
	// in order to do so, we want to use separate client for accessing k8s resources
	// because, client provided by controller manager are scoped only for operator namespace ¯\_(ツ)_/¯
	cl, err := client.New(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
		Mapper: mgr.GetRESTMapper(),
	})
	if err != nil {
		setupLog.Error(err, "unable to create uncached client")
		os.Exit(1)
	}

	if err = cdpipeline.NewReconcileCDPipeline(
		cl,
		mgr.GetScheme(),
		argocd.NewArgoApplicationSetManager(cl).CreateApplicationSet,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "cd-pipeline")
		os.Exit(1)
	}

	ctrlLog := ctrl.Log.WithName("controllers")

	if err = stage.NewReconcileStage(
		cl,
		mgr.GetScheme(),
		ctrlLog,
		objectmodifier.NewStageBatchModifierAll(cl, mgr.GetScheme()),
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "cd-stage")
		os.Exit(1)
	}

	if err = clustersecret.NewReconcileClusterSecret(cl, newAwsTokenGenerator()).
		SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "cluster-secret")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhook.RegisterValidationWebHook(context.Background(), mgr, ns); err != nil {
			setupLog.Error(err, "failed to create webhook")
			os.Exit(1)
		}
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")

	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func newAwsTokenGenerator() aws.AIMAuthTokenGenerator {
	g, err := aws.NewTokenGenerator()
	if err != nil {
		setupLog.Error(err, "unable to create aws token generator")
		os.Exit(1)
	}

	return g
}
