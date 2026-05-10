import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '@/api'
import { isValidRating, type RatingDimension } from '@stuhelper/shared/course'
import type { ReviewRatings } from '@stuhelper/shared/review'
import { localizeRatingDimension } from '@/modules/review/ratingHelpers'

export function useRatingDimensions() {
  const { t } = useI18n()
  const rawDimensions = ref<RatingDimension[]>([])
  const loading = ref(false)
  const loadFailed = ref(false)

  const dimensions = computed(() =>
    [...rawDimensions.value]
      .filter((dimension) => dimension.isActive)
      .sort((left, right) => left.sortOrder - right.sortOrder)
      .map((dimension) => localizeRatingDimension(dimension, t)),
  )

  const ratingKeys = computed(() => dimensions.value.map((dimension) => dimension.key))
  const ratingKeySet = computed<ReadonlySet<string>>(() => new Set(ratingKeys.value))

  async function load() {
    loading.value = true
    loadFailed.value = false
    try {
      const response = await api.rating.getDimensions()
      rawDimensions.value = response.data?.data ?? []
    } catch (_error) { void _error;
      rawDimensions.value = []
      loadFailed.value = true
    } finally {
      loading.value = false
    }
  }

  onMounted(() => {
    void load()
  })

  return {
    dimensions,
    ratingKeys,
    ratingKeySet,
    loading,
    loadFailed,
    load,
  }
}

export function areRatingsComplete(
  ratings: ReviewRatings,
  dimensions: ReadonlyArray<Pick<RatingDimension, 'key'>>,
): boolean {
  return dimensions.length > 0 && dimensions.every((dimension) => isValidRating(ratings[dimension.key]))
}

export function filterRatingsByDimensions(
  ratings: ReviewRatings,
  ratingKeys: ReadonlySet<string>,
): ReviewRatings {
  const filteredEntries = Object.entries(ratings).filter(([key, value]) => ratingKeys.has(key) && isValidRating(value))
  return Object.fromEntries(filteredEntries) as ReviewRatings
}
