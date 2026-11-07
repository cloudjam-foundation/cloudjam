<!-- Button toggling the sidebar -->
<script lang="ts">
	import PanelLeftIcon from '@lucide/svelte/icons/panel-left';
	import Button from '../Button.svelte';
	import { useSidebar } from './state.svelte';

	let { class: className }: { class?: string } = $props();

	const sidebar = useSidebar();
</script>

<Button variant="ghost" class={className} title="Toggle Sidebar" aria-label="Toggle Sidebar" onclick={sidebar.toggle}>
	<PanelLeftIcon class="size-4" />
</Button>
