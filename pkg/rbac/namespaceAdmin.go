package rbac

import (
	"fmt"
	rbacV1 "k8s.io/api/rbac/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const adminRoleNamePattern = "%s:admin"

type NamespaceAdminRole struct {
	BaseRole
	K8sClient kubernetes.Interface
}

func NewNamespaceAdminRole(namespace string, k8sclient kubernetes.Interface) (role *NamespaceAdminRole) {
	role = &NamespaceAdminRole{}
	role.Namespace = namespace
	role.RoleName = generateAdminRoleName(namespace)
	role.K8sClient = k8sclient
	return
}

func (role *NamespaceAdminRole) CreateRole() error {
	adminRoleName := generateAdminRoleName(role.Namespace)
	_, err := role.K8sClient.RbacV1().Roles(role.Namespace).Get(adminRoleName, metaV1.GetOptions{})
	if err != nil {
		switch t := err.(type) {
		case *k8sError.StatusError:
			if t.Status().Reason == metaV1.StatusReasonNotFound {
				roleTmp := &rbacV1.Role{}
				roleTmp.APIVersion = "v1"
				roleTmp.APIVersion = "v1"
				roleTmp.Kind = "Role"
				roleTmp.Name = adminRoleName
				roleTmp.Namespace = role.Namespace
				roleTmp.Rules = append(roleTmp.Rules,
					rbacV1.PolicyRule{
						APIGroups: []string{"*"},
						Resources: []string{"*"},
						Verbs:     []string{"*"}},
				)
				_, err := role.K8sClient.RbacV1().Roles(role.Namespace).Create(roleTmp)
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

func (role *NamespaceAdminRole) CreateRoleBinding(accountNamespace, accountName string) error {
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

func generateAdminRoleName(namespace string) string {
	return fmt.Sprintf(adminRoleNamePattern, namespace)
}
