import { Glue, Submit, type SubmitState } from '$lib';
import { GetRequestSchema } from '$lib/sdk/v1/admin/user/user_pb';
import type { User } from '$lib/sdk/v1/admin/user_pb';
import { createState } from '$lib/utils.svelte';
import { create } from '@bufbuild/protobuf';
import type { PageLoad } from './$types';

export const prerender = 'auto';
export const trailingSlash = 'always';
export const ssr = false;

export const load: PageLoad = async ({ params }) => {
	let user = createState<User | undefined>(undefined);
	let userState = createState<SubmitState>({ error: '', forbidden: false, loading: false });

	const refresh = async () => {
		await Submit(async () => {
			user.value = (await Glue.user.get(create(GetRequestSchema, { id: params.id }))).user;
		}, userState.value);
	};
	await refresh();

	return {
		get user() {
			return user.value;
		},
		get userState() {
			return userState.value;
		},
		refresh
	};
};
