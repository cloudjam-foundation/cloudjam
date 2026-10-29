<script lang="ts">
	import * as Sidebar from '$lib/components/shad/sidebar';
	import * as Popover from '$lib/components/shad/popover';
	import * as Field from '$lib/components/shad/field';
	import { useSidebar } from '$lib/components/shad/sidebar';
	import Avatar from '$lib/components/custom/sidebar/Avatar.svelte';
	import { Skeleton } from '$lib/components/shad/skeleton';
	import { ChevronsUpDownIcon, CogIcon, SunIcon, MoonIcon, LoaderCircleIcon, LogOutIcon } from '@lucide/svelte';
	import { mode, resetMode, setMode } from 'mode-watcher';
	import { toast } from 'svelte-sonner';
	import { setToken } from '$lib';
	import { goto } from '$app/navigation';
	import Button from '../buttons/Button.svelte';

	const sidebar = useSidebar();
	let { pubId, email, loading = $bindable(true) }: { pubId: string; email: string; loading: boolean } = $props();

	let open = $state(false);
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

			toast.success(`Theme changed!`);
		} catch (e) {
			toast.error('Unable to change theme! Please report this as a bug!');
			console.error(e);
		}
		themeChanging = false;
	}

	async function toggleTheme() {
		await setTheme(mode.current === 'light' ? 'dark' : 'light');
	}
</script>

<Sidebar.MenuItem>
	<Popover.Root bind:open>
		<Popover.Trigger disabled={loading}>
			{#snippet child({ props })}
				<Sidebar.MenuButton
					{...props}
					size="lg"
					class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
				>
					{#if loading}
						<div
							class="text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg"
						>
							<Skeleton class="h-8 w-8 rounded-full" />
						</div>
						<div class="grid flex-1 text-start text-sm leading-tight">
							<span class="truncate font-medium">
								<Skeleton class="h-6 w-full" />
							</span>
						</div>
					{:else}
						<div
							class="text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg"
						>
							<Avatar {pubId} name={email} width="6" height="6" />
						</div>
						<div class="grid flex-1 text-start text-sm leading-tight">
							<span>{email}</span>
						</div>
					{/if}

					<ChevronsUpDownIcon class="ms-auto" />
				</Sidebar.MenuButton>
			{/snippet}
		</Popover.Trigger>

		<Popover.Content
			align="start"
			class="w-(--bits-dropdown-menu-anchor-width) min-w-48 rounded-lg"
			side={sidebar.isMobile ? 'top' : 'right'}
			sideOffset={15}
		>
			<Popover.Header class="flex flex-col items-center">
				<Avatar {pubId} name={email} width="w-20" height="h-20" />
				<Popover.Title class="inline-flex items-center text-2xl font-bold">
					{email}
				</Popover.Title>
				<!-- <Popover.Description>{email}</Popover.Description> -->
			</Popover.Header>

			<Field.Group>
				<Field.Content>
					<Field.Group>
						<Field.Field orientation="horizontal" class="w-full">
							<Field.Label>Theme</Field.Label>
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
									<SunIcon class="" />
								{:else}
									<MoonIcon class="" />
								{/if}
							</Button>
						</Field.Field>
					</Field.Group>
				</Field.Content>
			</Field.Group>

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
		</Popover.Content>
	</Popover.Root>
</Sidebar.MenuItem>
