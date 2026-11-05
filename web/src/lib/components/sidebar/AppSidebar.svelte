<script lang="ts">
	import SectionRenderer from '$lib/components/sidebar/options/SectionRenderer.svelte';
	import Profile from '$lib/components/sidebar/Profile.svelte';
	import SidebarController from '$lib/components/sidebar/SidebarController.svelte';
	import Sidebar from './Sidebar.svelte';
	import { adminOptions, hostOptions, jamOptions, providerOptions } from './sidebarOptions';
	import { Feature } from '$lib/sdk/v1/admin/role_pb';
	import type { User } from '$lib/sdk/v1/admin/user_pb';

	let { user, userIcon, features }: { user: User | undefined; userIcon: string; features: Feature[] } = $props();
</script>

<Sidebar class="z-100 dense-glass" {header} {footer}>
	{#if features.includes(Feature.VisualJam)}
		<SectionRenderer section={jamOptions} />
	{/if}
	{#if features.includes(Feature.VisualHost)}
		<SectionRenderer section={hostOptions} />
	{/if}
	{#if features.includes(Feature.VisualProvider)}
		<SectionRenderer section={providerOptions} />
	{/if}
	{#if features.includes(Feature.VisualAdmin)}
		<SectionRenderer section={adminOptions} />
	{/if}
	{#if features.includes(Feature.VisualLeaderboard)}
		<!-- <SectionRenderer section={hallOfFameOptions} /> -->
	{/if}
</Sidebar>

{#snippet header()}
	<SidebarController />
{/snippet}

{#snippet footer()}
	<Profile {user} {userIcon} />
{/snippet}
