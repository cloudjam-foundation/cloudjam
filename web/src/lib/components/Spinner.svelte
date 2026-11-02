<script lang="ts">
	import { cn } from '$lib/utils';
	import { LoaderIcon } from '@lucide/svelte';

	let { class: className }: { class?: string } = $props();
</script>

<LoaderIcon role="status" aria-label="Loading" class={cn('size-4 animate-spin', className)} />
