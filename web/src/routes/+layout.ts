import { getSubject } from '$lib';
import { loadActive } from '$lib/auth.svelte';
import { redirect } from '@sveltejs/kit';
import type { LayoutLoad } from './$types';

export const prerender = 'auto';
export const trailingSlash = 'always';
export const ssr = false;

export const load: LayoutLoad = async ({ url }) => {
	if (!getSubject()) {
		// Since trailing slash is enabled, this must be included in the path check.
		if (url.pathname !== '/login/' && url.pathname !== '/register/') {
			redirect(307, '/login/');
		}
	}
	await loadActive();
};
