# useAsyncData Migration Design

**Date**: 2026-03-04
**Status**: Approved
**Author**: Code Review Follow-up

## Problem

18 components contain duplicate loading/error handling logic:

```typescript
const loading = ref(false)
const error = ref(false)
const data = ref(null)

async function fetch() {
  loading.value = true
  try {
    const res = await api()
    data.value = res.data
  } catch (e) {
    error.value = true
  } finally {
    loading.value = false
  }
}
```

This pattern repeats across the codebase, causing maintenance burden.

## Solution

Use `useAsyncData` composable to unify async state management.

### Current Implementation

Already created at `src/composables/useAsyncData.ts`:
- ✅ Supports parameterized fetchers via closures
- ✅ Provides manual `execute()` trigger
- ✅ Supports `immediate` option
- ✅ Auto-manages loading/error/data states

### Migration Strategy

**Phase 1: Simple Components (8 components)**

Migrate components with single data source and no complex logic:
- AuthCallbackPage
- DashboardPage
- LogsPage
- ReportsPage
- SensitiveWordsPage
- TeachersManagePage
- MyFavoritesTab
- MyVotesTab

**Phase 2: Complex Components (10 components)**

Keep current implementation for:
- Multi-source data loading (CourseDetailPage)
- Infinite scroll (ReviewFeed)
- Optimistic updates (FavoriteButton)
- Search with debounce (CommandPalette)

These components have optimized logic that doesn't benefit from migration.

## Implementation

### Before
```typescript
const loading = ref(true)
const stats = ref<AdminStats>({ /* defaults */ })

async function fetchStats() {
  loading.value = true
  try {
    const res = await getAdminStats()
    stats.value = res.data
  } finally {
    loading.value = false
  }
}

onMounted(fetchStats)
```

### After
```typescript
const { data: stats, loading, error, execute } = useAsyncData(
  () => getAdminStats().then(res => res.data)
)
```

**Benefits**:
- 10+ lines → 3 lines
- Auto error handling
- Consistent pattern across codebase

## Testing

For each migrated component:
1. Verify loading state displays correctly
2. Verify data loads on mount
3. Verify manual refresh works
4. Verify error handling (if applicable)

## Rollout

1. Migrate one component
2. Test thoroughly
3. Migrate remaining 7 components
4. Update archiving.md
