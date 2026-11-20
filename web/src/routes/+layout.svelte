<script lang="ts">
	import './layout.css';
	import black from '$lib/assets/black.png';
	import white from '$lib/assets/white.png';
	import { mode, ModeWatcher } from 'mode-watcher';
	import AppSidebar from '$lib/components/sidebar/AppSidebar.svelte';
	import { getActiveFeatures, getActiveUser, getActiveUserIcon } from '$lib/auth.svelte';
	import { page } from '$app/state';
	import Background from '$lib/components/ui/Background.svelte';
	import { onNavigate } from '$app/navigation';

	let { children } = $props();

	onNavigate((navigation) => {
		if (navigation.from?.route.id !== '/login') return; // not from login, skip animation
		if (!document.startViewTransition) return; // unsupported browser, skip animation
		return new Promise((resolve) => {
			document.startViewTransition(async () => {
				resolve();
				await navigation.complete;
			});
		});
	});
</script>

<svelte:head><link rel="icon" href={mode.current === 'dark' ? black : white} /></svelte:head>

<ModeWatcher />

<div class="relative isolate min-h-screen w-full overflow-hidden">
	<Background />
	{#if page.route.id !== '/login' && page.route.id !== '/register'}
		<div class="flex min-h-svh w-full">
			<AppSidebar user={getActiveUser()} userIcon={getActiveUserIcon()} features={getActiveFeatures()} />

			<main class="overflow-scroll-hidden h-dvh min-w-0 flex-1">
				<section class="h-full min-w-0 p-3 sm:p-5">
					{@render children?.()}
				</section>
			</main>
		</div>
	{:else}
		<main class="overflow-scroll-hidden h-dvh min-w-0 flex-1">
			<section class="h-full min-w-0 p-3 sm:p-5">
				{@render children?.()}
			</section>
		</main>
	{/if}
</div>
