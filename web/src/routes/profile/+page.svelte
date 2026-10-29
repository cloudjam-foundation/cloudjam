<script lang="ts">
	import { toSvg } from 'jdenticon';
	import { create } from '@bufbuild/protobuf';
	import { Glue, setToken, Submit, type SubmitState } from '$lib';
	import { onMount } from 'svelte';
	import { timestampFromDate } from '@bufbuild/protobuf/wkt';
	import { GetRequestSchema, ResetPasswordRequestSchema, UpdateRequestSchema } from '$lib/sdk/v1/admin/user/user_pb';
	import { UserSchema, type User } from '$lib/sdk/v1/admin/user_pb';
	import Gauge from './Gauge.svelte';
	import { goto } from '$app/navigation';
	import { CircleCheck, CircleX, LogOut, Pencil, ShieldCheck } from '@lucide/svelte';
	import Input from '$lib/components/shad/input/input.svelte';
	import * as Dialog from '$lib/components/shad/dialog';
	import Card from '$lib/components/layout/Card.svelte';
	import { resolve } from '$app/paths';
	import Button from '$lib/components/custom/buttons/Button.svelte';
	import LinkButton from '$lib/components/custom/buttons/LinkButton.svelte';
	import StaticBadge from '$lib/components/custom/badges/StaticBadge.svelte';
	import Alert from '$lib/components/custom/Alert.svelte';

	let user: User | undefined = $state();

	let userState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let updateState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let resetState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let edit = $state(false);

	onMount(async () => {
		Submit(async () => {
			user = (await Glue.user.get(create(GetRequestSchema, {}))).user;
		}, userState);
	});
</script>

<svelte:head>
	<title>{user?.username ?? 'Profile'} | CloudJam</title>
	<meta property="og:title" content="Profile | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full flex-col items-center justify-center gap-4">
	{#if user}
		<Card class="my-[5%] w-full lg:flex-row">
			{#if edit}
				<form
					class="flex h-full w-full flex-col items-start gap-1"
					onsubmit={() =>
						Submit(async () => {
							await Glue.user.update(create(UpdateRequestSchema, { mod: user }));
							edit = false;
						}, updateState)}
				>
					<div class="mt-2 flex h-full w-full flex-col items-start gap-4">
						<h1 class="text-4xl opacity-80">Edit User</h1>
						<div class="flex flex-col gap-1">
							<label class="text-sm">
								Description
								<Input bind:value={user.username} placeholder="Change your username" />
							</label>
							<p class="text-xs text-destructive">{Glue.Validate(UserSchema, user).violation.username}</p>
						</div>
						<div class="flex flex-col gap-1">
							<label class="text-sm">
								Your Slogan (use \n for newlines)
								<Input bind:value={user.description} placeholder="Invent a creative Slogan" />
							</label>
							<p class="text-xs text-destructive">{Glue.Validate(UserSchema, user).violation.description}</p>
						</div>
						<div class="mt-auto flex flex-row items-center gap-2">
							<Button variant="primary" onclick={() => (edit = false)}>
								<CircleX />
								Close
							</Button>
							<Button
								type="submit"
								processing={updateState.loading}
								variant="primary"
								disabled={Boolean(Glue.Validate(UpdateRequestSchema, create(UpdateRequestSchema, { mod: user })).error)}
							>
								<CircleCheck />
								Update
							</Button>
						</div>

						{#if updateState.error}
							<Alert title="Update failed" message={updateState.error} animation={true} />
						{/if}
					</div>
				</form>
				<div class="h-1 w-full rounded-2xl bg-neutral/80 lg:h-64 lg:w-1"></div>
				<div class="flex w-full flex-col items-start justify-center gap-8 lg:flex-row">
					<Dialog.Root>
						<Dialog.Trigger>
							<Button variant="primary">Change Password</Button>
						</Dialog.Trigger>
						<Dialog.Content>
							<Dialog.Header>
								<Dialog.Title>Reset password?</Dialog.Title>
								<Dialog.Description>This will take you back to the registration process.</Dialog.Description>
							</Dialog.Header>
							<Button
								processing={resetState.loading}
								onclick={() => {
									Submit(async () => {
										const resp = await Glue.user.resetPassword(
											create(ResetPasswordRequestSchema, {
												email: user?.email,
												expires: timestampFromDate(new Date(Date.now() + 10 * 60 * 1000)) // expires in 10 minutes
											})
										);
										goto(resolve(`/register?email=${user!.email}&username=${user!.username}&code=${resp.code}`));
									}, resetState);
								}}
								variant="destructive"
							>
								Yes, proceed
							</Button>

							{#if resetState.error}
								<Alert title="Failed to reset" message={resetState.error} animation={true} />
							{/if}
						</Dialog.Content>
					</Dialog.Root>
				</div>
			{:else}
				<div class="flex w-full flex-col items-start gap-1">
					<img
						alt="user profile"
						src={`data:image/svg+xml;base64,${btoa(toSvg(user.pubId, 140))}`}
						height="8rem"
						class="rounded-lg bg-primary/5"
					/>
					<h1 class="text-4xl opacity-80">{user.username}</h1>
					<p class="mt-auto text-neutral/80">
						Proud CloudJamer since {new Date(Number(user.createdAt) * 1000).toLocaleDateString()}
					</p>
					<StaticBadge text={user.organization} variant="filled" shimmer={true} class="font-bold" />
					<div class="mt-2 flex flex-row items-center gap-2">
						<Button variant="primary" onclick={() => (edit = true)}>
							<Pencil />
							Edit
						</Button>
						<LinkButton href="/admin">
							<ShieldCheck />
							Admin
						</LinkButton>
						<Button
							variant="destructive"
							onclick={() => {
								setToken('');
								goto(resolve('/login'));
							}}
						>
							<LogOut />
							Logout
						</Button>
					</div>
				</div>
				<div class="h-1 w-full rounded-2xl bg-neutral/80 lg:h-64 lg:w-1"></div>
				<div class="flex w-full flex-col items-center justify-end gap-8 lg:flex-row">
					<!-- TODO add chart -->
					<Gauge title="Score" scale={30} center={user.score} outer={user.maxScore} inner={user.score} />
					<Gauge title="Streak" scale={1} center={user.streak} outer={user.maxStreak} inner={user.streak} />
				</div>
			{/if}
		</Card>
	{/if}
	<h1 class="w-full glass-text py-3 text-start text-9xl font-bold">
		{#each user?.description.split('\\n') as line}
			{line}<br />
		{/each}
	</h1>
	{#if userState.error}
		<Alert title="Failed to load user" message={userState.error} animation={true} />
	{/if}
</div>
