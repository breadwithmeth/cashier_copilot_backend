import { Injectable,OnModuleInit } from '@nestjs/common'; import { PrismaClient } from '@cashier/database';
@Injectable() export class PrismaService extends PrismaClient implements OnModuleInit {
 async onModuleInit(){await this.$connect()}
 get user():any{return this.users} get refreshToken():any{return this.refresh_tokens}
 get analyticsWorker():any{return this.analytics_workers} get workerCameraAssignment():any{return this.worker_camera_assignments}
 get camera():any{return this.cameras} get cameraRoi():any{return this.camera_rois} get store():any{return this.stores}
 get rule():any{return this.rules} get shift():any{return this.shifts} get mlModel():any{return this.models}
 get analyticsEvent():any{return this.analytics_events} get eventType():any{return this.event_types}
 get notification():any{return this.notifications} get eventReview():any{return this.event_reviews}
 get processingSession():any{return this.processing_sessions} get cameraMetric():any{return this.camera_metrics}
 get eventEvidence():any{return this.event_evidence}
}
