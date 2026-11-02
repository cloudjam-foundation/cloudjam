<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Glue, Submit, type SubmitState } from '$lib';
	import logo from '$lib/assets/favicon.svg';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import Card from '$lib/components/Card.svelte';
	import { RegisterRequestSchema } from '$lib/sdk/v1/auth/auth_pb';
	import { create } from '@bufbuild/protobuf';
	import { onMount } from 'svelte';

	let registerRequest = $state(create(RegisterRequestSchema, {}));

	let registerState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	onMount(() => {
		registerRequest.email = page.url.searchParams.get('email') ?? '';
		registerRequest.username = page.url.searchParams.get('username') ?? '';
		registerRequest.code = page.url.searchParams.get('code') ?? '';
		history.replaceState(null, '', '/register');
	});
</script>

<svelte:head>
	<title>Register | CloudJam</title>
	<meta property="og:title" content="RegisteRegister | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full items-center justify-center">
	<Card class="mt-[10%] w-96">
		<form
			class="flex w-full flex-col items-center gap-4"
			onsubmit={() =>
				Submit(async () => {
					await Glue.auth.register(registerRequest);
					goto('/login');
				}, registerState)}
		>
			<img alt="icon" src={logo} class="h-32" />
			<h1 class="text-4xl opacity-80">Registration</h1>
			<Input bind:value={registerRequest.email} class="w-full" placeholder="Please enter your Email" type="email" />
			<Input
				bind:value={registerRequest.code}
				class="w-full"
				placeholder="Please enter your registration code"
				type="text"
			/>
			<hr class="w-full border border-neutral/40" />
			<Input
				bind:value={registerRequest.username}
				class="w-full"
				placeholder="Create a creative username"
				type="text"
			/>
			<Input
				bind:value={registerRequest.password}
				class="w-full"
				placeholder="Create a supersecret password"
				type="password"
			/>
			<Input
				bind:value={registerRequest.confirmPassword}
				class="w-full"
				placeholder="Confirm your supersecret password"
				type="password"
			/>
			<Button
				class="w-full"
				type="submit"
				variant="primary"
				processing={registerState.loading}
				disabled={Boolean(Glue.Validate(RegisterRequestSchema, registerRequest).error)}
			>
				Register
			</Button>
			{#if registerState.error}
				<Alert title="Registration failed" message={registerState.error} animation={true} />
			{/if}
		</form>
	</Card>
</div>
