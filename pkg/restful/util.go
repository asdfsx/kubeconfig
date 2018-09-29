package restful

import "fmt"


const readOnlyRole = "cluster-readonly"
const roleBindingPattern = "%s:%s:%s-binding"

func getRoleBindingName(namespace, serviceAccount, role string) string {
	return fmt.Sprintf(roleBindingPattern, namespace, serviceAccount, role)
}
