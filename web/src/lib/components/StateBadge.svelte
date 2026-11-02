<!-- Badge with built-in mappings to opinionated states that represent a color & animation -->
<script lang="ts">
	type State =
		| 'StableNeutral'
		| 'StablePositive'
		| 'StableWarning'
		| 'StableNegative'
		| 'RunningNeutral'
		| 'RunningPositive'
		| 'RunningWarning'
		| 'RunningNegative';

	let {
		text,
		state,
		mapper
	}:
		| {
				text: string;
				state: string;
				mapper: Record<string, State>;
		  }
		| {
				text: string;
				state: State;
				mapper: undefined;
		  } = $props();

	let finalState = $derived(mapper ? mapper[state] : state);
</script>

<div class="glass w-max rounded-full px-3 py-[0.2rem] text-center text-xs">
	{#if finalState === 'StableNeutral'}
		<span class="text-primary/70"> {text} </span>
	{:else if finalState === 'StablePositive'}
		<span class="text-emerald-500"> {text} </span>
	{:else if finalState === 'StableWarning'}
		<span class="text-orange-600"> {text} </span>
	{:else if finalState === 'StableNegative'}
		<span class="text-red-500"> {text} </span>
	{:else if finalState === 'RunningNeutral'}
		<span class="shimmer shimmer-color-primary/70"> {text} </span>
	{:else if finalState === 'RunningPositive'}
		<span class="shimmer text-emerald-500 shimmer-color-emerald-800"> {text} </span>
	{:else if finalState === 'RunningWarning'}
		<span class="shimmer text-orange-400 shimmer-color-orange-800"> {text} </span>
	{:else if finalState === 'RunningNegative'}
		<span class="shimmer text-red-500 shimmer-color-red-900"> {text} </span>
	{/if}
</div>
