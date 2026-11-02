import { Glue, Submit, type SubmitState } from '$lib';
import { GetRequestSchema } from '$lib/sdk/v1/play/game/game_pb';
import type { Game } from '$lib/sdk/v1/play/game_pb';
import { createState } from '$lib/utils.svelte';
import { create } from '@bufbuild/protobuf';
import type { PageLoad } from './$types';

export const prerender = 'auto';
export const trailingSlash = 'always';
export const ssr = false;

export const load: PageLoad = async ({ params }) => {
	let game = createState<Game | undefined>(undefined);
	let gameState = createState<SubmitState>({ error: '', forbidden: false, loading: false });

	const refresh = async () => {
		await Submit(async () => {
			game.value = (await Glue.game.get(create(GetRequestSchema, { id: params.id }))).game;
		}, gameState.value);
	};
	await refresh();

	return {
		get game() {
			return game.value;
		},
		get gameState() {
			return gameState.value;
		},
		refresh
	};
};
