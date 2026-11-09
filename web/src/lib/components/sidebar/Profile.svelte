<script lang="ts">
	import { goto } from '$app/navigation';
	import { setToken } from '$lib';
	import { ChevronsUpDownIcon, CogIcon, LoaderCircleIcon, LogOutIcon, MoonIcon, SunIcon } from '@lucide/svelte';
	import { mode, resetMode, setMode } from 'mode-watcher';
	import Button from '../Button.svelte';
	import SidebarButton from './SidebarButton.svelte';
	import { useSidebar } from './state.svelte';
	import type { User } from '$lib/sdk/v1/admin/user_pb';
	import { resolve } from '$app/paths';

	const sidebar = useSidebar();
	let { user, userIcon }: { user: User | undefined; userIcon: string } = $props();

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
	<SidebarButton size="lg" disabled={!user} onclick={() => (open = !open)} class={open ? 'bg-foreground/5' : ''}>
		{#if !user}
			<div class="flex aspect-square size-8 items-center justify-center rounded-lg">
				<div class="h-8 w-8 animate-pulse rounded-full bg-muted"></div>
			</div>
			<div class="grid flex-1 text-start text-sm leading-tight">
				<span class="truncate font-medium">
					<span class="block h-6 w-full animate-pulse rounded-md bg-muted"></span>
				</span>
			</div>
		{:else}
			<img alt="profile-icon" src={userIcon} class="aspect-square size-5 rounded-full object-cover" />
			<div class="grid flex-1 text-start text-sm leading-tight">
				<span>{user.email}</span>
			</div>
		{/if}

		<ChevronsUpDownIcon class="ms-auto" />
	</SidebarButton>

	{#if open}
		<div
			class="absolute flex w-72 min-w-48 animate-in flex-col gap-4 rounded-lg bg-background p-4 text-sm shadow-md ring-1 ring-foreground/10 duration-100 fade-in-0 {sidebar.isMobile
				? 'bottom-full left-0 mb-4'
				: 'bottom-0 left-full ml-4'}"
		>
			<div class="flex flex-col items-center gap-2">
				<img alt="profile-icon" src={userIcon} class="aspect-square size-20 rounded-full object-cover" />
				<h1 class="inline-flex items-center text-center text-xl font-bold break-all">
					{user?.username}
				</h1>
			</div>

			<div class="flex w-full flex-row items-center justify-between gap-2">
				<span class="text-sm font-medium">Theme</span>
				<Button
					variant="primary"
					class="ml-auto"
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
					goto(resolve('/profile'));
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
					goto(resolve('/login'));
				}}
				variant="destructive"
			>
				<LogOutIcon />
				Logout
			</Button>
		</div>
	{/if}
</div>
