<script lang="ts">
	import SidebarButton from '$lib/components/sidebar/SidebarButton.svelte';
	import type { sidebarOption } from '$lib/components/sidebar/sidebarOptions';

	let { option }: { option: sidebarOption } = $props();
</script>

<li class="relative list-none">
	<SidebarButton href={option.link}>
		{@const Icon = option.icon}
		<Icon class="mr-2 h-4 w-4" />
		<span>{option.title}</span>
	</SidebarButton>
</li>
