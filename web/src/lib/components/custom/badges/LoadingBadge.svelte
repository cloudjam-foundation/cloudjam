<!-- Badge with loading animation and spinner -->
<script lang="ts">
	import Spinner from '$lib/components/shad/spinner/spinner.svelte';

	let { text }: { text: string } = $props();
</script>

<div
	class="glass flex w-max flex-row items-center justify-center gap-1 rounded-full px-2 py-[0.2rem] text-center text-xs"
>
	<span class="shimmer text-primary shimmer-color-muted-foreground"> {text} </span>
	<Spinner class="size-3" />
</div>

<style>
	.glass {
		box-shadow:
			rgba(255, 255, 255, 0.05) 0px 3px 6px 0px,
			rgba(255, 255, 255, 0.08) 0px 0px 0px 1px;
		background-color: rgba(255, 255, 255, 0.01);
		backdrop-filter: blur(2px);
	}
</style>
