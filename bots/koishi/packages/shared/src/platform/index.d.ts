import type { StuhelperPlatformConfig } from '../types';
export interface PlatformClient {
    getHealth(): Promise<void>;
}
export declare function createPlatformClient(config: StuhelperPlatformConfig): PlatformClient;
