<script lang="ts">
	import { onMount } from "svelte";
	import { Sparkles, TrendingUp, Zap } from "@lucide/svelte";
	import HeroLearningCard from "$lib/components/dashboard/HeroLearningCard.svelte";
	import GiaCoachCard from "$lib/components/dashboard/GiaCoachCard.svelte";
	import DeadlineCard from "$lib/components/dashboard/DeadlineCard.svelte";
	import LearningPathTimeline from "$lib/components/dashboard/LearningPathTimeline.svelte";
	import AchievementPanel from "$lib/components/dashboard/AchievementPanel.svelte";
	import SkillRadarCard from "$lib/components/dashboard/SkillRadarCard.svelte";
	import RecommendationList from "$lib/components/dashboard/RecommendationList.svelte";
	import LeaderboardCard from "$lib/components/dashboard/LeaderboardCard.svelte";
	import LatestNewsCard from "$lib/components/dashboard/LatestNewsCard.svelte";
	import type {
		AchievementItem,
		DeadlineItem,
		LeaderboardItem,
		NewsItem,
	} from "$lib/components/dashboard/types";

	let { data } = $props();

	let userName = $derived((data as any)?.user?.name ?? "Learner");
	let firstName = $derived(userName.split(/\s+/)[0] || "Learner");
	let totalScore = $derived((data as any)?.gamification?.totalScore ?? 0);
	let dashboard = $derived((data as any)?.dashboard ?? {});

	let resumeHref = $derived(
		dashboard.continue_learning?.id
			? `/lesson-player?id=${dashboard.continue_learning.id}`
			: "/lesson-player",
	);

	let deadlines = $derived<DeadlineItem[]>(
		(dashboard.deadlines ?? []).map((item: any) => ({
			title: item.title,
			due: item.due,
			daysLeft: item.days_left,
			isOverdue: item.is_overdue,
			courseId: item.course_id,
		})),
	);

	let achievements = $derived<AchievementItem[]>(
		(dashboard.achievements ?? []).map((item: any) => ({
			label: item.label,
			tier: item.tier,
			icon: Zap,
		})),
	);

	const newsIcons: Record<string, typeof Sparkles> = {
		sparkles: Sparkles,
		"trending-up": TrendingUp,
	};
	let news = $derived<NewsItem[]>(
		(dashboard.whats_new ?? []).map((item: any) => ({
			title: item.title,
			meta: item.meta,
			icon: newsIcons[item.icon] ?? Sparkles,
			href: item.href ?? item.url ?? (item.course_id ? `/lesson-player?id=${item.course_id}` : undefined),
		})),
	);

	let leaderboard = $state<LeaderboardItem[]>([]);
	let leaderboardLoading = $state(true);

	onMount(() => {
		leaderboard = [{ name: userName, xp: totalScore, me: true }];
		const controller = new AbortController();
		(async () => {
			try {
				const response = await fetch("/api/gamification/leaderboard?limit=10", {
					credentials: "include",
					signal: controller.signal,
				});
				if (response.ok) {
					const payload = await response.json();
					leaderboard = (payload.leaderboard ?? []).map((row: any) => ({
						name: row.name,
						xp: row.total_score,
						me: row.is_me,
					}));
				}
			} catch {
				// Keep the signed-in learner visible when live leaderboard data is unavailable.
			} finally {
				leaderboardLoading = false;
			}
		})();
		return () => controller.abort();
	});

	let skillLabels = $derived(dashboard.skill_radar?.labels ?? []);
	let skillValues = $derived(
		dashboard.skill_radar?.data?.length === skillLabels.length
			? dashboard.skill_radar.data
			: [],
	);
</script>

<svelte:head>
	<title>Dashboard · FGB Academy</title>
	<meta name="description" content="Your FGB Academy learning dashboard." />
</svelte:head>

<div class="mx-auto flex w-full max-w-[1600px] flex-col gap-6">
	<div class="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,2.1fr)_minmax(280px,.95fr)]">
		<HeroLearningCard {firstName} {resumeHref} />
		<div class="flex min-w-0 flex-col gap-4">
			<GiaCoachCard />
			<DeadlineCard deadline={deadlines[0]} />
		</div>
	</div>

	<LearningPathTimeline steps={dashboard.learning_path ?? []} />

	<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-[1.05fr_1.05fr_1.15fr]">
		<AchievementPanel {achievements} />
		<SkillRadarCard labels={skillLabels} values={skillValues} />
		<RecommendationList items={dashboard.recommended ?? []} />
	</div>

	<div class="grid grid-cols-1 gap-4 lg:grid-cols-[1.05fr_1.3fr]">
		<LeaderboardCard rows={leaderboard} loading={leaderboardLoading} />
		<LatestNewsCard items={news} />
	</div>
</div>
