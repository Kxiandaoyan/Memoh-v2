import { nextTick, watch, type Ref } from 'vue'
import { useElementBounding } from '@vueuse/core'
import { onBeforeRouteLeave } from 'vue-router'

/** Pixels from bottom to still count as "at bottom" for auto-scroll. */
const AT_BOTTOM_THRESHOLD = 240

/**
 * Auto-scroll: scroll to bottom when content grows; pause when user scrolls up, resume when near bottom.
 * Caches scroll position on route leave.
 */
export function useAutoScroll(
  containerRef: Ref<HTMLElement | undefined>,
  loading: Ref<boolean>,
) {
  const { height, top } = useElementBounding(containerRef)

  let prevScroll = 0
  let curScroll = 0
  let autoScroll = true
  let cachedScroll = 0

  function getScrollParent() {
    let el = containerRef.value?.parentElement
    while (el) {
      const style = getComputedStyle(el)
      if (style.overflowY === 'auto' || style.overflowY === 'scroll') {
        return el
      }
      el = el.parentElement ?? undefined
    }
    return undefined
  }

  watch(top, () => {
    const container = getScrollParent()
    if (!container) return

    if (height.value === 0) {
      autoScroll = false
      prevScroll = curScroll = 0
    }

    const distanceToBottom = container.scrollHeight - container.clientHeight - container.scrollTop
    if (distanceToBottom <= AT_BOTTOM_THRESHOLD) {
      autoScroll = true
      prevScroll = curScroll = container.scrollTop
    }
  })

  watch(height, (newVal, oldVal) => {
    const container = getScrollParent()
    if (!container) return

    curScroll = container.scrollTop
    if (!loading.value && curScroll < prevScroll) {
      autoScroll = false
    }
    prevScroll = curScroll

    const isRestoringCached = cachedScroll > 0
    const isInitialContent = !isRestoringCached && newVal > container.clientHeight && (oldVal === 0 || (oldVal != null && newVal > oldVal * 1.5))
    if (isInitialContent) {
      nextTick(() => {
        const targetTop = isRestoringCached ? cachedScroll : container.scrollHeight - container.clientHeight
        container.scrollTo({ top: Math.max(0, targetTop), behavior: 'auto' })
      })
      return
    }

    const distanceToBottom = container.scrollHeight - container.clientHeight - container.scrollTop
    const contentGrew = newVal > (oldVal ?? 0)
    const shouldScrollToBottom = (loading.value && contentGrew) || (distanceToBottom <= AT_BOTTOM_THRESHOLD && autoScroll && contentGrew)
    if (shouldScrollToBottom) {
      nextTick(() => {
        const targetTop = container.scrollHeight - container.clientHeight
        container.scrollTo({
          top: targetTop,
          behavior: loading.value ? 'auto' : 'smooth',
        })
      })
    }
  })

  onBeforeRouteLeave(() => {
    const container = getScrollParent()
    if (container) {
      cachedScroll = container.scrollTop
    }
  })

  function scrollToBottom(behavior: ScrollBehavior = 'smooth') {
    const container = getScrollParent()
    if (!container) return
    container.scrollTo({
      top: container.scrollHeight - container.clientHeight,
      behavior,
    })
  }

  return { containerRef, getScrollParent, scrollToBottom }
}
