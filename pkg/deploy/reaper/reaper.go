package reaper

import (
	"time"

	"github.com/golang/glog"
	kapi "k8s.io/kubernetes/pkg/api"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/kubectl"

	"github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/deploy/util"
)

// NewDeploymentConfigReaper returns a new reaper for deploymentConfigs
func NewDeploymentConfigReaper(oc *client.Client, kc *kclient.Client) kubectl.Reaper {
	return &DeploymentConfigReaper{oc: oc, kc: kc, pollInterval: kubectl.Interval, timeout: kubectl.Timeout}
}

// DeploymentConfigReaper implements the Reaper interface for deploymentConfigs
type DeploymentConfigReaper struct {
	oc                    client.Interface
	kc                    kclient.Interface
	pollInterval, timeout time.Duration
}

// Stop scales a replication controller via its deployment configuration down to
// zero replicas, waits for all of them to get deleted and then deletes both the
// replication controller and its deployment configuration.
func (reaper *DeploymentConfigReaper) Stop(namespace, name string, timeout time.Duration, gracePeriod *kapi.DeleteOptions) error {
	// If the config is already deleted, it may still have associated
	// deployments which didn't get cleaned up during prior calls to Stop. If
	// the config can't be found, still make an attempt to clean up the
	// deployments.
	//
	// It's important to delete the config first to avoid an undesirable side
	// effect which can cause the deployment to be re-triggered upon the
	// config's deletion. See https://github.com/openshift/origin/issues/2721
	// for more details.
	err := reaper.oc.DeploymentConfigs(namespace).Delete(name)
	configNotFound := kerrors.IsNotFound(err)
	if err != nil && !configNotFound {
		return err
	}

	// Clean up deployments related to the config.
	options := kapi.ListOptions{LabelSelector: util.ConfigSelector(name)}
	rcList, err := reaper.kc.ReplicationControllers(namespace).List(options)
	if err != nil {
		return err
	}
	rcReaper, err := kubectl.ReaperFor(kapi.Kind("ReplicationController"), reaper.kc)
	if err != nil {
		return err
	}

	// If there is neither a config nor any deployments, nor any deployer pods, we can return NotFound.
	deployments := rcList.Items

	if configNotFound && len(deployments) == 0 {
		return kerrors.NewNotFound(kapi.Resource("deploymentconfig"), name)
	}
	for _, rc := range deployments {
		if err = rcReaper.Stop(rc.Namespace, rc.Name, timeout, gracePeriod); err != nil {
			// Better not error out here...
			glog.Infof("Cannot delete ReplicationController %s/%s: %v", rc.Namespace, rc.Name, err)
		}

		// Only remove deployer pods when the deployment was failed. For completed
		// deployment the pods should be already deleted.
		if !util.IsFailedDeployment(&rc) {
			continue
		}

		// Delete all deployer and hook pods
		options = kapi.ListOptions{LabelSelector: util.DeployerPodSelector(rc.Name)}
		podList, err := reaper.kc.Pods(rc.Namespace).List(options)
		if err != nil {
			return err
		}
		for _, pod := range podList.Items {
			err := reaper.kc.Pods(pod.Namespace).Delete(pod.Name, gracePeriod)
			if err != nil {
				// Better not error out here...
				glog.Infof("Cannot delete Pod %s/%s: %v", pod.Namespace, pod.Name, err)
			}
		}
	}

	return nil
}
