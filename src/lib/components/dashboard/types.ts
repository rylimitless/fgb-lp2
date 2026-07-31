import type { Component } from "svelte";

export type IconComponent = Component<{ class?: string; "aria-hidden"?: string }>;

export type DeadlineItem = {
	title: string;
	due: string;
	daysLeft: number | null;
	isOverdue: boolean;
	courseId: number;
};

export type LearningStep = {
	label: string;
	status: "Completed" | "In Progress" | "Not Started" | string;
};

export type RecommendationItem = {
	id?: number;
	title: string;
	duration?: string;
	level?: string;
	image?: string;
};

export type LeaderboardItem = {
	name: string;
	xp: number;
	me?: boolean;
};

export type AchievementItem = {
	label: string;
	tier: "gold" | "silver" | "bronze";
	icon?: IconComponent;
};

export type NewsItem = {
	title: string;
	meta: string;
	icon?: IconComponent;
	href?: string;
};
