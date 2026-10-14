export function migrateLegacyAdminStorage(
  storage: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> | undefined,
): void;
