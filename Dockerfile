FROM public.ecr.aws/lambda/go:1.x

COPY app ${LAMBDA_TASK_ROOT}

CMD [ "app" ]
