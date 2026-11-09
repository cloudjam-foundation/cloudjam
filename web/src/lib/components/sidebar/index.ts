import { browser } from '$app/env';

export function isMacOS(): boolean {
	if (!browser) return false;
	const platform = navigator?.platform?.toLowerCase() ?? '';
	const userAgent = navigator?.userAgent?.toLowerCase() ?? '';
	return platform.includes('mac') || userAgent.includes('mac');
}
