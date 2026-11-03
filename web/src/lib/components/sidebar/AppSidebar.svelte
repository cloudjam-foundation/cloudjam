<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import { getEmail, getPubId } from '$lib';
	import SectionRenderer from '$lib/components/sidebar/options/SectionRenderer.svelte';
	import Profile from '$lib/components/sidebar/Profile.svelte';
	import SidebarController from '$lib/components/sidebar/SidebarController.svelte';
	import { onMount } from 'svelte';
	import Sidebar from './Sidebar.svelte';
	import { adminOptions, hostOptions, jamOptions, providerOptions } from './sidebarOptions';

	let loading = $state(true);

	onMount(() => {
		loading = false;
	});

	let pubId = $state('');
	let email = $state('');

	onMount(() => {
		pubId = getPubId();
		email = getEmail();
	});

	afterNavigate(() => {
		pubId = getPubId();
		email = getEmail();
	});
</script>

<Sidebar class="z-100 dense-glass" {header} {footer}>
	<SectionRenderer section={jamOptions} />
	<SectionRenderer section={hostOptions} />
	<SectionRenderer section={providerOptions} />
	<SectionRenderer section={adminOptions} />
	<!-- <SectionRenderer section={hallOfFameOptions} /> -->
</Sidebar>

{#snippet header()}
	<SidebarController />
{/snippet}

{#snippet footer()}
	<Profile bind:loading {pubId} {email} />
{/snippet}
