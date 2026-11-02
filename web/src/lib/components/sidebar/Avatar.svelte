<script lang="ts">
	import WrappedTooltip from '$lib/components/WrappedTooltip.svelte';
	import { cn } from '$lib/utils';
	import { toSvg } from 'jdenticon';

	let {
		pubId,
		name,
		height,
		width
	}: {
		pubId: string;
		name: string;
		height?: string;
		width?: string;
	} = $props();

	let failed = $state(false);

	/**
	 * Convert a longer string into initials of a set length
	 * @param value The string to initialise
	 * @param length Optional, a length to cap the initials ot
	 * @returns A string that has been initialised
	 */
	function toShortInitials(value: string, length: number = 2) {
		if (value.length <= length) {
			return value.toUpperCase();
		}

		return value
			.split(' ')
			.join('')
			.substring(0, length - 1)
			.toUpperCase();
	}
</script>

<WrappedTooltip caption={name}>
	<span
		class={cn(
			'relative flex shrink-0 overflow-hidden rounded-full select-none after:absolute after:inset-0 after:rounded-full after:border after:border-border',
			height ?? 'h-12',
			width ?? 'w-12'
		)}
	>
		{#if failed}
			<span class="flex size-full items-center justify-center rounded-full bg-muted text-sm text-muted-foreground">
				{toShortInitials(name)}
			</span>
		{:else}
			<img
				alt={name}
				src={`data:image/svg+xml;base64,${btoa(toSvg(pubId, 140))}`}
				class="aspect-square size-full rounded-full object-cover"
				onerror={() => (failed = true)}
			/>
		{/if}
	</span>
</WrappedTooltip>
