/**
 * Routes index
 */

import { Router, IRouter } from 'express';
import authRoutes from './auth';
import projectRoutes from './projects';
import secretRoutes from './secrets';

const router: IRouter = Router();

router.use('/auth', authRoutes);
router.use('/projects', projectRoutes);
router.use('/projects', secretRoutes); // /projects/:slug/secrets routes

export default router;

