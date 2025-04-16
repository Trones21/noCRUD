import logging

logger = logging.getLogger("django.request")


class RequestResponseLoggerMiddleware:
    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        logger.info(f"Incoming Request: {request.method} {request.path}")
        response = self.get_response(request)
        logger.info(f"Outgoing Response: {response.status_code}")
        return response
