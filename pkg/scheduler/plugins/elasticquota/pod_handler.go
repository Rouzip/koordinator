/*
Copyright 2022 The Koordinator Authors.

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

package elasticquota

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// todo the eventHandler's operation should be a complete transaction in the future work.

func (g *Plugin) OnPodAdd(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}

	quotaName, treeID := g.getPodAssociateQuotaNameAndTreeID(pod)
	if quotaName == "" {
		return
	}

	mgr := g.GetGroupQuotaManagerForTree(treeID)
	if mgr != nil {
		mgr.OnPodAdd(quotaName, pod)
		klog.V(5).Infof("OnPodAddFunc %v add success, quota: %v, tree: [%v]", klog.KObj(pod), quotaName, treeID)
	} else {
		klog.Warningf("OnPodAddFunc %v add failed, quota: %v, quota manager not found: %v", klog.KObj(pod), quotaName, treeID)
	}
}

func (g *Plugin) OnPodUpdate(oldObj, newObj interface{}) {
	oldPod := oldObj.(*corev1.Pod)
	newPod := newObj.(*corev1.Pod)

	if oldPod.ResourceVersion == newPod.ResourceVersion {
		return
	}

	oldQuotaName, oldTree := g.getPodAssociateQuotaNameAndTreeID(oldPod)
	newQuotaName, newTree := g.getPodAssociateQuotaNameAndTreeID(newPod)

	if oldTree == newTree {
		mgr := g.GetGroupQuotaManagerForTree(newTree)
		if mgr != nil {
			if oldQuotaName == "" {
				if newQuotaName != "" {
					mgr.OnPodAdd(newQuotaName, newPod)
					klog.V(5).Infof("OnPodUpdateFunc %v add success, quota: %v, tree: [%v]", klog.KObj(newPod), newQuotaName, newTree)
				}
			} else {
				if newQuotaName != "" {
					mgr.OnPodUpdate(newQuotaName, oldQuotaName, newPod, oldPod)
					klog.V(5).Infof("OnPodUpdateFunc %v update success, quota:%v, tree: [%v]", klog.KObj(newPod), newQuotaName, newTree)
				} else {
					mgr.OnPodDelete(oldQuotaName, oldPod)
					klog.V(5).Infof("OnPodUpdateFunc %v delete success, quota:%v, tree: [%v]", klog.KObj(oldPod), oldQuotaName, oldTree)
				}
			}
		} else {
			klog.Errorf("OnPodUpdateFunc %v update failed, quota: %v, quota manager not found: %v", klog.KObj(newPod), newQuotaName, newTree)
		}
		return
	}

	oldMgr := g.GetGroupQuotaManagerForTree(oldTree)
	newMgr := g.GetGroupQuotaManagerForTree(newTree)
	if oldMgr != nil {
		if oldQuotaName != "" {
			oldMgr.OnPodDelete(oldQuotaName, oldPod)
			klog.V(5).Infof("OnPodUpdateFunc %v, delete success, quota: %v, tree: %v", klog.KObj(oldPod), oldQuotaName, oldTree)
		}
	} else {
		klog.Errorf("OnPodUpdateFunc %v delete failed, quota: %v, quota manager not found: %v", klog.KObj(oldPod), oldQuotaName, oldTree)
	}
	if newMgr != nil {
		if newQuotaName != "" {
			newMgr.OnPodAdd(newQuotaName, newPod)
			klog.V(5).Infof("OnPodUpdateFunc %v add success, quota: %v, tree: %v ", klog.KObj(newPod), newQuotaName, newTree)
		}
	} else {
		klog.Errorf("OnPodUpdateFunc %v add failed, quota: %v, quota manager not found: %v", klog.KObj(newPod), newQuotaName, newTree)
	}
}

func (g *Plugin) OnPodDelete(obj interface{}) {
	var pod *corev1.Pod
	switch t := obj.(type) {
	case *corev1.Pod:
		pod = t
	case cache.DeletedFinalStateUnknown:
		pod, _ = t.Obj.(*corev1.Pod)
	}
	if pod == nil {
		klog.V(4).InfoS("OnPodDeleteFunc, failed to parse object, obj: %T", obj)
		return
	}

	g.handlePodDelete(pod)
}

func (g *Plugin) handlePodDelete(pod *corev1.Pod) {
	quotaName, treeID := g.getPodAssociateQuotaNameAndTreeID(pod)
	if quotaName == "" {
		return
	}

	mgr := g.GetGroupQuotaManagerForTree(treeID)
	if mgr != nil {
		mgr.OnPodDelete(quotaName, pod)
		klog.V(5).Infof("OnPodDeleteFunc %v delete success, quota: %v, tree: %v", klog.KObj(pod), quotaName, treeID)
	} else {
		klog.Errorf("OnPodDeleteFunc %v delete failed, quota: %v, tree: %v", klog.KObj(pod), quotaName, treeID)
	}
}
