<!-- Badge with loading animation and spinner -->
<script lang="ts">
	import Spinner from '$lib/components/Spinner.svelte';

	let { text }: { text: string } = $props();
</script>

<div
	class="flex w-max flex-row items-center justify-center gap-1 rounded-full glass px-2 py-[0.2rem] text-center text-xs"
>
	<span class="shimmer text-primary shimmer-color-muted-foreground"> {text} </span>
	<Spinner class="size-3" />
</div>
