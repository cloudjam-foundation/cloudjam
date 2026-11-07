import { Glue, Submit, type SubmitState } from '$lib';
import { GetRequestSchema } from '$lib/sdk/v1/admin/role/role_pb';
import type { Role } from '$lib/sdk/v1/admin/role_pb';
import { createState } from '$lib/utils.svelte';
import { create } from '@bufbuild/protobuf';
import type { PageLoad } from './$types';

export const prerender = 'auto';
export const trailingSlash = 'always';
export const ssr = false;

export const load: PageLoad = async ({ params }) => {
	// TODO: Figure out how to get these to stop throwing ESLint errors.
	let role = createState<Role | undefined>(undefined);
	let roleState = createState<SubmitState>({ error: '', forbidden: false, loading: false });

	const refresh = async () => {
		await Submit(async () => {
			role.value = (await Glue.role.get(create(GetRequestSchema, { id: params.id }))).role;
		}, roleState.value);
	};
	await refresh();

	return {
		get role() {
			return role.value;
		},
		get roleState() {
			return roleState.value;
		},
		refresh
	};
};
