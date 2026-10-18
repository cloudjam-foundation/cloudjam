<script lang="ts">
	let { text, shimmer }: { text: string; shimmer?: boolean } = $props();
</script>

<div class="glass w-max rounded-full px-3 py-[0.2rem] text-center text-xs">
	<span class="text-primary {shimmer ? 'shimmer shimmer-color-muted-foreground' : ''}"> {text} </span>
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
