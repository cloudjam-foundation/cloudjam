import { Glue, Submit, type SubmitState } from '$lib';
import { GetRequestSchema } from '$lib/sdk/v1/cloud/provider/provider_pb';
import type { Provider } from '$lib/sdk/v1/cloud/provider_pb';
import { createState } from '$lib/utils.svelte';
import { create } from '@bufbuild/protobuf';
import type { PageLoad } from './$types';

export const prerender = 'auto';
export const trailingSlash = 'always';
export const ssr = false;

export const load: PageLoad = async ({ params }) => {
	let provider = createState<Provider | undefined>(undefined);
	let providerState = createState<SubmitState>({ error: '', forbidden: false, loading: false });

	const refresh = async () => {
		await Submit(async () => {
			provider.value = (await Glue.provider.get(create(GetRequestSchema, { id: params.id }))).provider;
		}, providerState.value);
	};
	await refresh();

	return {
		get provider() {
			return provider.value;
		},
		get providerState() {
			return providerState.value;
		},
		refresh
	};
};
