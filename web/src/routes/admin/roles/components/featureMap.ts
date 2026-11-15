import { Feature } from '$lib/sdk/v1/admin/role_pb';

type FeatureInternal = {
	id: Feature.VisualJam | Feature.VisualAdmin | Feature.VisualHost | Feature.VisualLeaderboard | Feature.VisualProvider;
	title: string;
	description: string;
};

export const featureMap: FeatureInternal[] = [
	{
		id: 0,
		title: 'Games',
		description: 'Allow users of this role to view and compete in games.'
	},
	{
		id: 1,
		title: 'Leaderboard',
		description: 'Allow users of this role to view the standing of competitors.'
	},
	{
		id: 2,
		title: 'Host',
		description: 'Allow users of this role to view and manage games and challenges.'
	},
	{
		id: 3,
		title: 'Providers',
		description: 'Allow users of this role to view and manage providers that games run on.'
	},
	{
		id: 4,
		title: 'Admin',
		description: 'Allow users of this role to view and manage users, role, tenants and the system.'
	}
];
