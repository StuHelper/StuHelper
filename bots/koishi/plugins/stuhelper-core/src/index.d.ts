import { Context, Schema } from 'koishi';
import { type StuhelperCoreConfig } from '@stuhelper/koishi-shared';
export declare const name = "stuhelper-core";
export type Config = StuhelperCoreConfig;
export declare const Config: Schema<Config>;
export declare function apply(ctx: Context, config: Config): void;
