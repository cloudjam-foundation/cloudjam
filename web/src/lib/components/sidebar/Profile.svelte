<script lang="ts">
	import { goto } from '$app/navigation';
	import { setToken } from '$lib';
	import Avatar from '$lib/components/sidebar/Avatar.svelte';
	import { ChevronsUpDownIcon, CogIcon, LoaderCircleIcon, LogOutIcon, MoonIcon, SunIcon } from '@lucide/svelte';
	import { mode, resetMode, setMode } from 'mode-watcher';
	import Button from '../Button.svelte';
	import SidebarButton from './SidebarButton.svelte';
	import { useSidebar } from './state.svelte';

	const sidebar = useSidebar();
	let { pubId, email, loading = $bindable(true) }: { pubId: string; email: string; loading: boolean } = $props();

	let open = $state(false);
	let root: HTMLDivElement | undefined = $state();
	let themeChanging = $state(false);

	// Theme Stuff
	type ThemeMode = 'light' | 'dark' | 'system';

	async function setTheme(nextTheme: ThemeMode) {
		if (themeChanging || mode.current === nextTheme) return;

		themeChanging = true;
		try {
			switch (nextTheme) {
				case 'light':
				case 'dark':
					setMode(nextTheme);
					break;
				default:
					resetMode();
					break;
			}
		} catch (e) {
			console.error(e);
		}
		themeChanging = false;
	}

	async function toggleTheme() {
		await setTheme(mode.current === 'light' ? 'dark' : 'light');
	}
</script>

<svelte:window
	onpointerdown={(e) => {
		if (open && root && !root.contains(e.target as Node)) open = false;
	}}
	onkeydown={(e) => {
		if (open && e.key === 'Escape') open = false;
	}}
/>

<div bind:this={root} class="relative">
	<SidebarButton size="lg" disabled={loading} onclick={() => (open = !open)} class={open ? 'bg-foreground/5' : ''}>
		{#if loading}
			<div class="flex aspect-square size-8 items-center justify-center rounded-lg">
				<div class="h-8 w-8 animate-pulse rounded-full bg-muted"></div>
			</div>
			<div class="grid flex-1 text-start text-sm leading-tight">
				<span class="truncate font-medium">
					<span class="block h-6 w-full animate-pulse rounded-md bg-muted"></span>
				</span>
			</div>
		{:else}
			<div class="flex aspect-square size-8 items-center justify-center rounded-lg">
				<Avatar {pubId} name={email} width="w-6" height="h-6" />
			</div>
			<div class="grid flex-1 text-start text-sm leading-tight">
				<span>{email}</span>
			</div>
		{/if}

		<ChevronsUpDownIcon class="ms-auto" />
	</SidebarButton>

	{#if open}
		<div
			class="absolute z-50 flex w-72 min-w-48 animate-in flex-col gap-4 rounded-lg bg-background p-4 text-sm shadow-md ring-1 ring-foreground/10 duration-100 fade-in-0 zoom-in-95 {sidebar.isMobile
				? 'bottom-full left-0 mb-4'
				: 'bottom-0 left-full ml-4'}"
		>
			<div class="flex flex-col items-center gap-2">
				<Avatar {pubId} name={email} width="w-20" height="h-20" />
				<h1 class="inline-flex items-center text-center text-2xl font-bold break-all">
					{email}
				</h1>
			</div>

			<div class="flex w-full flex-row items-center gap-2">
				<span class="text-sm font-medium">Theme</span>
				<Button
					variant="primary"
					class="mr-auto ml-auto"
					onclick={async () => await toggleTheme()}
					disabled={themeChanging}
					aria-busy={themeChanging}
				>
					{#if themeChanging}
						<LoaderCircleIcon class="animate-spin" />
					{:else if mode.current === 'light'}
						<SunIcon />
					{:else}
						<MoonIcon />
					{/if}
				</Button>
			</div>

			<Button
				onclick={() => {
					goto('/profile');
					sidebar.setOpen(false);
					open = false;
				}}
				variant="primary"
			>
				<CogIcon />
				Settings
			</Button>

			<Button
				onclick={() => {
					setToken('');
					goto('/login');
				}}
				variant="destructive"
			>
				<LogOutIcon />
				Logout
			</Button>
		</div>
	{/if}
</div>
