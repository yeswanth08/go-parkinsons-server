/* generated using openapi-typescript-codegen -- do not edit */
/* istanbul ignore file */
/* tslint:disable */
/* eslint-disable */
import type { ParkinsonsResponse } from '../models/ParkinsonsResponse';
import type { CancelablePromise } from '../core/CancelablePromise';
import { OpenAPI } from '../core/OpenAPI';
import { request as __request } from '../core/request';
export class DefaultService {
    /**
     * Detect Parkinsons via WebSocket audio stream
     * @returns ParkinsonsResponse Analysis result (returned as WS text frame)
     * @throws ApiError
     */
    public static detectWs({
        age,
        sex,
    }: {
        age: number,
        sex: 0 | 1,
    }): CancelablePromise<ParkinsonsResponse> {
        return __request(OpenAPI, {
            method: 'GET',
            url: '/api/v1/detect/ws',
            query: {
                'age': age,
                'sex': sex,
            },
            errors: {
                500: `Internal error`,
            },
        });
    }
    /**
     * @returns ParkinsonsResponse
     * @throws ApiError
     */
    public static detectUpload({
        age,
        sex,
        formData,
    }: {
        age: number,
        sex: number,
        formData: {
            audio?: Blob;
        },
    }): CancelablePromise<ParkinsonsResponse> {
        return __request(OpenAPI, {
            method: 'POST',
            url: '/api/v1/detect/upload',
            query: {
                'age': age,
                'sex': sex,
            },
            formData: formData,
            mediaType: 'multipart/form-data',
        });
    }
}
