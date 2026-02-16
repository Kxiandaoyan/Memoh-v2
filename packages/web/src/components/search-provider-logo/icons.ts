/**
 * Search provider icon registry (FontAwesome).
 *
 * To add a new provider icon:
 * 1. Find the icon in FontAwesome (https://fontawesome.com/icons)
 * 2. Import it in `main.ts` and add to `library.add()`
 * 3. Add the [prefix, iconName] tuple to PROVIDER_ICONS below
 *
 * The key must match the `provider` field stored in the database (lowercase).
 */

const PROVIDER_ICONS: Record<string, [string, string]> = {
  brave: ['fab', 'brave'],
}

const DEFAULT_ICON: [string, string] = ['fas', 'globe']

export function getSearchProviderIcon(provider: string): [string, string] {
  if (!provider) return DEFAULT_ICON
  return PROVIDER_ICONS[provider.trim().toLowerCase()] ?? DEFAULT_ICON
}
