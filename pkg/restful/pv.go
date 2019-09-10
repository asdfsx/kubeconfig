package restful

import (
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful-openapi"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

type persistentVolumeResource struct {
	k8sClient                kubernetes.Interface
	selfDefineResourcePrefix string
}

type persistentVolumeAction struct {
	PvName       string `json:"pvName" description:"name of the pv"`
	PvcName      string `json:"pvcName" description:"name of the pvc"`
	NameSpace    string `json:"namespace" description:"name of the namespace"`
	NfsPath      string `json:"nfsPath" description:"path of the nfs"`
	NfsIp        string `json:"nfsIp" description:"ip of the nfs"`
	StorageClass string `json:"storageClass" description:"class of the storage"`
	Storage      string `json:"storage" description:"quantity of the storage"`
	AccessMode   string `json:"accessMode" description:"mode of the access"`
}

type persistentVolumeEntity struct {
	pv  coreV1.PersistentVolume      `json:pv`
	pvc coreV1.PersistentVolumeClaim `json:pvc`
}

func createPersistVolumeResource(k8sClient kubernetes.Interface, prefix string) (resource *persistentVolumeResource) {
	resource = &persistentVolumeResource{
		k8sClient:                k8sClient,
		selfDefineResourcePrefix: prefix,
	}
	return
}

func (pvr persistentVolumeResource) WebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/pv").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	tags := []string{"pv"}

	ws.Route(ws.POST("/").To(pvr.createPersistentVolumeClaim).
		Doc("create shared pv").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(persistentVolumeAction{}). // on the response
		Returns(200, "OK", nil).
		Returns(404, "Not Found", nil))

	return ws
}

func (pvr persistentVolumeResource) createPersistentVolumeClaim(request *restful.Request, response *restful.Response) {

	action := &persistentVolumeAction{}
	if err := request.ReadEntity(action); err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	quantity, err := resource.ParseQuantity(action.Storage)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	_, err = pvr.k8sClient.CoreV1().PersistentVolumes().Get(action.PvName, metaV1.GetOptions{})
	if err == nil {
		response.Write([]byte("{\"status\":\"success\"}"))
	}

	accessMode := coreV1.PersistentVolumeAccessMode(action.AccessMode)
	nfs := coreV1.NFSVolumeSource{Server: action.NfsIp, Path: action.NfsPath}

	persistentVolumeTemp := &coreV1.PersistentVolume{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolume",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name: action.PvName,
		},
		Spec: coreV1.PersistentVolumeSpec{
			AccessModes:                   []coreV1.PersistentVolumeAccessMode{accessMode},
			Capacity:                      coreV1.ResourceList{coreV1.ResourceStorage: quantity},
			PersistentVolumeSource:        coreV1.PersistentVolumeSource{NFS: &nfs},
			PersistentVolumeReclaimPolicy: "Delete",
			StorageClassName:              action.StorageClass,
		},
	}

	persistentVolumeClaimTemp := &coreV1.PersistentVolumeClaim{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      action.PvcName,
			Namespace: action.NameSpace,
		},
		Spec: coreV1.PersistentVolumeClaimSpec{
			AccessModes:      []coreV1.PersistentVolumeAccessMode{accessMode},
			Resources:        coreV1.ResourceRequirements{Requests: coreV1.ResourceList{coreV1.ResourceStorage: quantity}},
			StorageClassName: &action.StorageClass,
			VolumeName:       action.PvName,
		},
	}

	persistentVolumeTemp, err = pvr.k8sClient.CoreV1().PersistentVolumes().Create(persistentVolumeTemp)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	persistentVolumeClaimTemp, err = pvr.k8sClient.CoreV1().PersistentVolumeClaims(action.NameSpace).Create(persistentVolumeClaimTemp)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(persistentVolumeEntity{*persistentVolumeTemp, *persistentVolumeClaimTemp})
	return

}
