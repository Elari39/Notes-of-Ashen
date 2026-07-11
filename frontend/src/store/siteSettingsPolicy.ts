export type PublicFeatureRouteState = 'loading' | 'error' | 'notFound' | 'content';

export interface SiteSettingsOperationState {
  isLoading: boolean;
  hasLoaded: boolean;
  loadError: string;
}

interface SiteSettingsOperationDependencies<
  TState extends SiteSettingsOperationState,
  TData,
> {
  request: () => Promise<TData>;
  setState: (patch: Partial<TState>) => void;
  toState: (data: TData) => Partial<TState>;
}

interface UpdateSiteSettingsOperationDependencies<
  TState extends SiteSettingsOperationState,
  TData,
> extends SiteSettingsOperationDependencies<TState, TData> {
  hasLoaded: boolean;
}

interface PublicFeatureRouteInput {
  hasLoaded: boolean;
  isLoading: boolean;
  loadError: string;
  enabled: boolean;
}

export const canWriteSiteSettings = (hasLoaded: boolean): boolean => hasLoaded;

export const areSiteSettingsControlsDisabled = (
  hasLoaded: boolean,
  isLoading: boolean,
): boolean => !hasLoaded || isLoading;

const operationPatch = <TState extends SiteSettingsOperationState>(
  patch: Partial<SiteSettingsOperationState>,
): Partial<TState> => patch as Partial<TState>;

export const executeFetchSettings = async <
  TState extends SiteSettingsOperationState,
  TData,
>({
  request,
  setState,
  toState,
}: SiteSettingsOperationDependencies<TState, TData>): Promise<void> => {
  setState(operationPatch<TState>({ isLoading: true, loadError: '' }));
  try {
    const data = await request();
    setState({
      ...toState(data),
      ...operationPatch<TState>({ hasLoaded: true, loadError: '' }),
    });
  } catch (error) {
    setState(operationPatch<TState>({
      hasLoaded: false,
      loadError: error instanceof Error ? error.message : 'Failed to load site settings',
    }));
  } finally {
    setState(operationPatch<TState>({ isLoading: false }));
  }
};

export const executeUpdateSettings = async <
  TState extends SiteSettingsOperationState,
  TData,
>({
  hasLoaded,
  request,
  setState,
  toState,
}: UpdateSiteSettingsOperationDependencies<TState, TData>): Promise<void> => {
  if (!canWriteSiteSettings(hasLoaded)) {
    throw new Error('site settings are not loaded yet');
  }

  setState(operationPatch<TState>({ isLoading: true }));
  try {
    const data = await request();
    setState({
      ...toState(data),
      ...operationPatch<TState>({ hasLoaded: true }),
    });
  } finally {
    setState(operationPatch<TState>({ isLoading: false }));
  }
};

export const resolvePublicFeatureRoute = ({
  hasLoaded,
  loadError,
  enabled,
}: PublicFeatureRouteInput): PublicFeatureRouteState => {
  if (!hasLoaded) {
    return loadError ? 'error' : 'loading';
  }

  return enabled ? 'content' : 'notFound';
};
