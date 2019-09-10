package rbac

import (
	rbacV1 "k8s.io/api/rbac/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const ReadOnlyRole = "cluster-readonly"

type ClusterReadonlyRole struct {
	BaseRole
	K8sClient kubernetes.Interface
}

func NewClusterReadonlyRoleRole(namespace, rolename string, k8sclient kubernetes.Interface) (role *ClusterReadonlyRole) {
	role = &ClusterReadonlyRole{}
	role.Namespace = namespace
	role.RoleName = rolename
	role.K8sClient = k8sclient
	return
}

func (role *ClusterReadonlyRole) CreateRole() error {
	_, err := role.K8sClient.RbacV1().ClusterRoles().Get(ReadOnlyRole, metaV1.GetOptions{})
	if err != nil {
		switch t := err.(type) {
		case *k8sError.StatusError:
			if t.Status().Reason == metaV1.StatusReasonNotFound {
				roleTmp := &rbacV1.ClusterRole{}
				roleTmp.APIVersion = "v1"
				roleTmp.Kind = "ClusterRole"
				roleTmp.Name = ReadOnlyRole
				roleTmp.Namespace = role.Namespace
				roleTmp.Rules = append(roleTmp.Rules,
					rbacV1.PolicyRule{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list", "watch"}},
					rbacV1.PolicyRule{
						APIGroups: []string{"batch"},
						Resources: []string{"jobs"},
						Verbs:     []string{"get", "list", "watch"}},
					rbacV1.PolicyRule{
						APIGroups: []string{""},
						Resources: []string{"services"},
						Verbs:     []string{"get", "list", "watch"}},
					rbacV1.PolicyRule{
						APIGroups: []string{"kubeflow.org"},
						Resources: []string{"tfjobs"},
						Verbs:     []string{"get", "list", "watch"}},
				)
				_, err := role.K8sClient.RbacV1().ClusterRoles().Create(roleTmp)
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

func (role *ClusterReadonlyRole) CreateRoleBinding(accountNamespace, accountName string) error {
	bindingName := GenerateRoleBindingName(role.RoleName, accountNamespace, accountName)
	_, err := role.K8sClient.RbacV1().ClusterRoleBindings().Get(bindingName, metaV1.GetOptions{})
	if err != nil {
		switch t := err.(type) {
		case *k8sError.StatusError:
			if t.Status().Reason == metaV1.StatusReasonNotFound {
				rolebindingtmp := &rbacV1.ClusterRoleBinding{}
				rolebindingtmp.APIVersion = "v1"
				rolebindingtmp.Kind = "ClusterRoleBinding"
				rolebindingtmp.Name = bindingName
				rolebindingtmp.Namespace = accountNamespace
				rolebindingtmp.Subjects = append(rolebindingtmp.Subjects, rbacV1.Subject{
					Kind:      "ServiceAccount",
					Name:      accountName,
					Namespace: accountNamespace,
				})
				rolebindingtmp.RoleRef.Kind = "ClusterRole"
				rolebindingtmp.RoleRef.Name = ReadOnlyRole

				_, err = role.K8sClient.RbacV1().ClusterRoleBindings().Create(rolebindingtmp)
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
