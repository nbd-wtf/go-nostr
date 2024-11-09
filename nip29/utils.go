package nip29

import "slices"

func (group Group) GetRoleByName(name string) *Role {
	idx := slices.IndexFunc(group.Roles, func(role *Role) bool { return role.Name == name })
	if idx == -1 {
		return &Role{Name: name}
	} else {
		return group.Roles[idx]
	}
}
