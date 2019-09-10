package rbac

import (
	rbacV1 "k8s.io/api/rbac/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type TillerRole struct {
	BaseRole
	K8sClient kubernetes.Interface
}

func NewTillerRole(namespace, rolename string, k8sclient kubernetes.Interface) (role *TillerRole) {
	role = &TillerRole{}
	role.Namespace = namespace
	role.RoleName = rolename
	role.K8sClient = k8sclient
	return role
}

// tiller 的 role 应该在安装helm的时候创建好，所以这里只是检查是否存在，不执行创建了
func (role *TillerRole) CreateRole() error {
	_, err := role.K8sClient.RbacV1().Roles(role.Namespace).Get(role.RoleName, metaV1.GetOptions{})
	return err
}

func (role *TillerRole) CreateRoleBinding(accountNamespace, accountName string) error {
	bindingName := GenerateRoleBindingName(role.RoleName, accountNamespace, accountName)
	_, err := role.K8sClient.RbacV1().RoleBindings(role.Namespace).Get(bindingName, metaV1.GetOptions{})
	if err != nil {
		switch t := err.(type) {
		case *k8sError.StatusError:
			if t.Status().Reason == metaV1.StatusReasonNotFound {
				rolebindingtmp := &rbacV1.RoleBinding{}
				rolebindingtmp.APIVersion = "v1"
				rolebindingtmp.Kind = "RoleBinding"
				rolebindingtmp.Name = bindingName
				rolebindingtmp.Namespace = role.Namespace
				rolebindingtmp.Subjects = append(rolebindingtmp.Subjects, rbacV1.Subject{
					Kind:      "ServiceAccount",
					Name:      accountName,
					Namespace: accountNamespace,
				})
				rolebindingtmp.RoleRef.Kind = "Role"
				rolebindingtmp.RoleRef.Name = role.RoleName

				_, err = role.K8sClient.RbacV1().RoleBindings(role.Namespace).Create(rolebindingtmp)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		default:
			return err
		}
	}

	return nil
}
