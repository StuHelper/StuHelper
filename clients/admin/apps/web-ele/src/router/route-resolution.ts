import type {
  LocationQueryValue,
  RouteLocationResolved,
  Router,
  RouteRecordRaw,
} from 'vue-router';

type NavigableMenu = { children?: NavigableMenu[]; path?: string };

function normalizeRoutePath(path: string) {
  if (!path || path === '/') {
    return '/';
  }
  return path.endsWith('/') ? path.slice(0, -1) : path;
}

function resolveRoutePath(parentPath: string, routePath: string) {
  if (routePath.startsWith('/')) {
    return normalizeRoutePath(routePath);
  }
  if (!parentPath || parentPath === '/') {
    return normalizeRoutePath(`/${routePath}`);
  }
  return normalizeRoutePath(`${parentPath}/${routePath}`);
}

function collectAbsoluteRoutePaths(routes: RouteRecordRaw[]) {
  const paths = new Set<string>();

  function visit(route: RouteRecordRaw, parentPath = '') {
    let currentPath = parentPath;
    if (typeof route.path === 'string') {
      currentPath = resolveRoutePath(parentPath, route.path);
      paths.add(currentPath);
    }
    route.children?.forEach((child) => visit(child, currentPath));
  }

  routes.forEach((route) => visit(route));
  return paths;
}

function collectRouteNames(routes: RouteRecordRaw[]) {
  const names = new Set<string>();

  function visit(route: RouteRecordRaw) {
    if (route.name !== undefined && route.name !== null) {
      names.add(String(route.name));
    }
    route.children?.forEach((child) => visit(child));
  }

  routes.forEach((route) => visit(route));
  return names;
}

function findFirstNavigablePath(menus: NavigableMenu[]): string | undefined {
  for (const menu of menus) {
    if (menu.children?.length) {
      const childPath = findFirstNavigablePath(menu.children);
      if (childPath) {
        return childPath;
      }
    }
    if (menu.path?.startsWith('/')) {
      return menu.path;
    }
  }
  return undefined;
}

function resolveHomePath(
  accessibleRoutes: RouteRecordRaw[],
  accessibleMenus: NavigableMenu[],
  defaultHomePath: string,
) {
  const accessiblePaths = collectAbsoluteRoutePaths(accessibleRoutes);
  if (accessiblePaths.has(normalizeRoutePath(defaultHomePath))) {
    return defaultHomePath;
  }
  return findFirstNavigablePath(accessibleMenus) ?? defaultHomePath;
}

function firstQueryValue(
  candidate: LocationQueryValue | LocationQueryValue[] | string | undefined,
) {
  if (Array.isArray(candidate)) {
    return candidate.find(
      (value): value is string => typeof value === 'string',
    );
  }
  return typeof candidate === 'string' ? candidate : undefined;
}

function decodeInternalPath(candidate: string | undefined) {
  if (!candidate) {
    return undefined;
  }
  try {
    const decoded = decodeURIComponent(candidate);
    return decoded.startsWith('/') && !decoded.startsWith('//')
      ? decoded
      : undefined;
  } catch {
    return undefined;
  }
}

function isAccessibleLocation(
  resolved: RouteLocationResolved,
  accessibleRouteNames: Set<string>,
  accessibleRoutePaths: Set<string>,
) {
  return resolved.matched.some((record) => {
    const routeName =
      record.name === undefined || record.name === null
        ? undefined
        : String(record.name);
    return (
      (routeName !== undefined && accessibleRouteNames.has(routeName)) ||
      accessibleRoutePaths.has(normalizeRoutePath(record.path))
    );
  });
}

function resolveAccessibleRedirect(
  router: Router,
  candidate: LocationQueryValue | LocationQueryValue[] | string | undefined,
  accessibleRoutes: RouteRecordRaw[],
  fallbackPath: string,
) {
  const fallback = router.resolve(fallbackPath);
  const decoded = decodeInternalPath(firstQueryValue(candidate));
  if (!decoded) {
    return fallback;
  }

  const resolved = router.resolve(decoded);
  const accessibleRouteNames = collectRouteNames(accessibleRoutes);
  const accessibleRoutePaths = collectAbsoluteRoutePaths(accessibleRoutes);
  return isAccessibleLocation(
    resolved,
    accessibleRouteNames,
    accessibleRoutePaths,
  )
    ? resolved
    : fallback;
}

export {
  collectAbsoluteRoutePaths,
  resolveAccessibleRedirect,
  resolveHomePath,
};
