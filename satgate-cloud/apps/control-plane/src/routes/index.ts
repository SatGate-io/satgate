/**
 * Routes index
 */

import { Router, IRouter } from 'express';
import authRoutes from './auth';
import projectRoutes from './projects';

const router: IRouter = Router();

router.use('/auth', authRoutes);
router.use('/projects', projectRoutes);

export default router;

