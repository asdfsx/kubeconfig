package rbac

import "fmt"

const roleBindingPattern = "%s:%s:%s-binding"

type BaseRole struct{
	Namespace    string
	RoleName     string
}

type RbacInterface interface{
	CreateRole() error
	CreateRoleBinding(accountNamespace, accountName string) error
}

func GenerateRoleBindingName(role, accountNamespace, accountName string) string {
	return fmt.Sprintf(roleBindingPattern, accountNamespace, accountName, role)
}