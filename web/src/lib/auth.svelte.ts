import { getSubject, Glue } from '$lib';
import { GetRequestSchema as GetRoleRequestSchema } from '$lib/sdk/v1/admin/role/role_pb';
import { GetRequestSchema as GetUserRequestSchema } from '$lib/sdk/v1/admin/user/user_pb';
import { create } from '@bufbuild/protobuf';
import { toSvg } from 'jdenticon';
import type { Feature, Role } from './sdk/v1/admin/role_pb';
import type { User } from './sdk/v1/admin/user_pb';

// constant identifier of the self scope (scope applies implicit to all resources you are considered the owner of).
export const SELF_SCOPE = 'self';
// constant identifier of the admin scope (scope that allows assignment of scopes you don't own).
export const ADMIN_SCOPE = 'admin';

let activeUser: User | undefined = $state();
// activeUser contains the current logged in user.
export function getActiveUser() {
	return activeUser;
}

let activeRole: Role | undefined = $state();
// activeRole contains the role of the current logged in user.
export function getActiveRole() {
	return activeRole;
}

// loadActive loads the activeUser/activeRole data from backend.
export async function loadActive() {
	if (!getSubject()) return;
	try {
		activeUser = (await Glue.user.get(create(GetUserRequestSchema, {}))).user;
	} catch {
		activeUser = undefined;
	}
	try {
		activeRole = (await Glue.role.get(create(GetRoleRequestSchema, { id: activeUser?.role }))).role;
	} catch {
		activeRole = undefined;
	}
}

// activeUserIcon contains a computed dataurl-svg derived from the logged in user public identifier.
const activeUserIcon: string = $derived(`data:image/svg+xml;base64,${btoa(toSvg(activeUser?.pubId, 512))}`);
export function getActiveUserIcon() {
	return activeUserIcon;
}

// activeFeatures contains all features the current logged in user.
const activeFeatures: Feature[] = $derived(activeRole?.features ?? []);
export function getActiveFeatures() {
	return activeFeatures;
}

// activeScopes contains all scopes the current logged in user has access to.
const activeScopes: string[] = $derived(Object.keys(activeRole?.permissions ?? []));
export function getActiveScopes() {
	return activeScopes;
}

// assignableTenants contains all activeScopes excluding self (aka tenants).
const assignableTenants: string[] = $derived(activeScopes.filter((scope) => scope !== SELF_SCOPE));
export function getAssignableTenants() {
	return assignableTenants;
}
