import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '@/api'
import { isValidRating, type RatingDimension } from '@stuhelper/shared/course'
import type { ReviewRatings } from '@stuhelper/shared/review'
import { localizeRatingDimension } from '@/modules/review/ratingHelpers'

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function readRatingDimension(payload: unknown): RatingDimension {
  if (!isRecord(payload)) {
    throw new Error('Invalid rating dimensions response')
  }

  const id = payload.id
  const key = payload.key
  const name = payload.name
  const description = payload.description
  const sortOrder = payload.sortOrder
  const isActive = payload.isActive
  const createdAt = payload.createdAt
  const updatedAt = payload.updatedAt
  const schoolID = payload.schoolID
  if (
    typeof id !== 'string' ||
    (schoolID !== undefined && (typeof schoolID !== 'number' || !Number.isInteger(schoolID))) ||
    typeof key !== 'string' ||
    typeof name !== 'string' ||
    (description !== undefined && typeof description !== 'string') ||
    typeof sortOrder !== 'number' ||
    !Number.isFinite(sortOrder) ||
    typeof isActive !== 'boolean' ||
    typeof createdAt !== 'string' ||
    typeof updatedAt !== 'string'
  ) {
    throw new Error('Invalid rating dimensions response')
  }

  return {
    id,
    schoolID,
    key,
    name,
    description,
    sortOrder,
    isActive,
    createdAt,
    updatedAt,
  }
}

function readRatingDimensionsPayload(payload: unknown): RatingDimension[] {
  if (!Array.isArray(payload)) {
    throw new Error('Invalid rating dimensions response')
  }

  return payload.map(readRatingDimension)
}

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
      rawDimensions.value = readRatingDimensionsPayload(response.data?.data)
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
