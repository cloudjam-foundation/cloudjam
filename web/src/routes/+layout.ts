import { getSubject } from '$lib';
import { loadActive } from '$lib/auth.svelte';
import { redirect } from '@sveltejs/kit';
import type { LayoutLoad } from './$types';

export const prerender = 'auto';
export const trailingSlash = 'always';
export const ssr = false;

export const load: LayoutLoad = async ({ url }) => {
	if (!getSubject()) {
		if (url.pathname !== '/login' && url.pathname !== '/register') {
			redirect(307, '/login');
		}
	}
	await loadActive();
};
